import * as jsonc from '@sqs/jsonc-parser'
import classnames from 'classnames'
import { camelCase } from 'lodash'
import React, { useCallback, useContext } from 'react'
import { Redirect } from 'react-router'
import { RouteComponentProps } from 'react-router-dom'

import { PlatformContextProps } from '@sourcegraph/shared/src/platform/context'
import { SettingsCascadeProps } from '@sourcegraph/shared/src/settings/settings'
import { asError } from '@sourcegraph/shared/src/util/errors'

import { AuthenticatedUser } from '../../../auth'
import { Page } from '../../../components/Page'
import { PageTitle } from '../../../components/PageTitle'
import { InsightsApiContext } from '../../core/backend/api-provider'
import { InsightTypeSuffix } from '../../core/types'

import {
    CreationSearchInsightForm,
    CreationSearchInsightFormProps,
} from './components/creation-search-insight-form/CreationSearchInsightForm'
import styles from './CreationSearchInsightPage.module.scss'
import { FORM_ERROR } from './hooks/useForm'

const defaultFormattingOptions: jsonc.FormattingOptions = {
    eol: '\n',
    insertSpaces: true,
    tabSize: 2,
}

export interface CreationSearchInsightPageProps
    extends PlatformContextProps<'updateSettings'>,
        Pick<RouteComponentProps, 'history'>,
        SettingsCascadeProps {
    /**
     * Authenticated user info, Used to decide where code insight will appears
     * in personal dashboard (private) or in organisation dashboard (public)
     * */
    authenticatedUser: Pick<AuthenticatedUser, 'id' | 'organizations'> | null
}

/** Displays create insight page with creation form. */
export const CreationSearchInsightPage: React.FunctionComponent<CreationSearchInsightPageProps> = props => {
    const { platformContext, authenticatedUser, history, settingsCascade } = props
    const { updateSubjectSettings, getSubjectSettings } = useContext(InsightsApiContext)

    const handleSubmit = useCallback<CreationSearchInsightFormProps['onSubmit']>(
        async values => {
            if (!authenticatedUser) {
                return
            }

            const {
                id: userID,
                organizations: { nodes: orgs },
            } = authenticatedUser
            const subjectID =
                values.visibility === 'personal'
                    ? userID
                    : // TODO [VK] Add org picker in creation UI and not just pick first organization
                      orgs[0].id

            try {
                const settings = await getSubjectSettings(subjectID).toPromise()

                const newSettingsString = {
                    title: values.title,
                    repositories: values.repositories.trim().split(/\s*,\s*/),
                    series: values.series.map(line => ({
                        name: line.name,
                        // Query field is a reg exp field for code insight query setting
                        // Native html input element adds escape symbols by itself
                        // to prevent this behavior below we replace double escaping
                        // with just one series of escape characters e.g. - //
                        query: line.query.replace(/\\\\/g, '\\'),
                        stroke: line.color,
                    })),
                    step: {
                        [values.step]: +values.stepValue,
                    },
                }

                const edits = jsonc.modify(
                    settings.contents,
                    // According to our naming convention <type>.insight.<name>
                    [`${InsightTypeSuffix.search}.${camelCase(values.title)}`],
                    newSettingsString,
                    { formattingOptions: defaultFormattingOptions }
                )

                const editedSettings = jsonc.applyEdits(settings.contents, edits)

                await updateSubjectSettings(platformContext, subjectID, editedSettings).toPromise()

                history.push('/insights')
            } catch (error) {
                return { [FORM_ERROR]: asError(error) }
            }

            return
        },
        [history, updateSubjectSettings, getSubjectSettings, platformContext, authenticatedUser]
    )

    const handleCancel = useCallback(() => {
        history.push('/insights')
    }, [history])

    // TODO [VK] Move this logic to high order component to simplify logic here
    if (authenticatedUser === null) {
        return <Redirect to="/" />
    }

    return (
        <Page className={classnames('col-8', styles.creationPage)}>
            <PageTitle title="Create new code insight" />

            <div className="mb-5">
                <h2>Create new code insight</h2>

                <p className="text-muted">
                    Search-based code insights analyze your code based on any search query.{' '}
                    <a
                        href="https://docs.sourcegraph.com/dev/background-information/insights"
                        target="_blank"
                        rel="noopener"
                    >
                        Learn more.
                    </a>
                </p>
            </div>

            <CreationSearchInsightForm
                className="pb-5"
                settings={settingsCascade.final}
                onSubmit={handleSubmit}
                onCancel={handleCancel}
            />
        </Page>
    )
}
