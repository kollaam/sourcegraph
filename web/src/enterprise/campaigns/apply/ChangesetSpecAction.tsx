import React from 'react'
import { ChangesetSpecType, ChangesetSpecFields } from '../../../graphql-operations'
import ClipboardCheckOutlineIcon from 'mdi-react/ClipboardCheckOutlineIcon'
import ClipboardAlertOutlineIcon from 'mdi-react/ClipboardAlertOutlineIcon'
import ClipboardArrowUpOutlineIcon from 'mdi-react/ClipboardArrowUpOutlineIcon'
import classNames from 'classnames'

export interface ChangesetSpecActionProps {
    spec: ChangesetSpecFields
    className?: string
}

export const ChangesetSpecAction: React.FunctionComponent<ChangesetSpecActionProps> = ({ spec, className }) => {
    if (spec.__typename === 'HiddenChangesetSpec') {
        if (spec.type === ChangesetSpecType.BRANCH) {
            return <ChangesetSpecActionNoPublish className={className} />
        }
        return <ChangesetSpecActionImport className={className} />
    }
    if (spec.description.__typename === 'ExistingChangesetReference') {
        return <ChangesetSpecActionImport className={className} />
    }
    if (spec.description.published) {
        return <ChangesetSpecActionPublish className={className} />
    }
    return <ChangesetSpecActionNoPublish className={className} />
}

const iconClassNames = 'm-0 text-nowrap d-block d-sm-flex flex-column align-items-center justify-content-center'

export const ChangesetSpecActionPublish: React.FunctionComponent<{ className?: string }> = ({ className }) => (
    <div className={classNames(className, iconClassNames)}>
        <ClipboardCheckOutlineIcon data-tooltip="This changeset will be published on the code host when the spec is applied." />
        <span className="text-muted">Will publish</span>
    </div>
)
export const ChangesetSpecActionNoPublish: React.FunctionComponent<{ className?: string }> = ({ className }) => (
    <div className={classNames(className, iconClassNames)}>
        <ClipboardAlertOutlineIcon data-tooltip="This changeset will NOT be published on the code host when the spec is applied." />
        <span className="text-muted">Won't publish</span>
    </div>
)
export const ChangesetSpecActionImport: React.FunctionComponent<{ className?: string }> = ({ className }) => (
    <div className={classNames(className, iconClassNames)}>
        <ClipboardArrowUpOutlineIcon data-tooltip="This changeset will be imported from the code host." />
        <span className="text-muted">Will import</span>
    </div>
)
