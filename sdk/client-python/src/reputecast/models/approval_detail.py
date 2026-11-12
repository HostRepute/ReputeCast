from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.approval_detail_channel import ApprovalDetailChannel
from ..models.approval_detail_status import ApprovalDetailStatus
from ..models.approval_detail_subject_type import ApprovalDetailSubjectType
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.approval_audience_view import ApprovalAudienceView
  from ..models.approval_audit_line import ApprovalAuditLine
  from ..models.approval_automation_snapshot import ApprovalAutomationSnapshot
  from ..models.approval_campaign_view import ApprovalCampaignView
  from ..models.approval_can import ApprovalCan
  from ..models.approval_change import ApprovalChange
  from ..models.approval_client_signoff_state import ApprovalClientSignoffState
  from ..models.approval_comment_view import ApprovalCommentView
  from ..models.approval_decision_view import ApprovalDecisionView
  from ..models.approval_page_snapshot import ApprovalPageSnapshot
  from ..models.approval_person import ApprovalPerson
  from ..models.approval_push_snapshot import ApprovalPushSnapshot
  from ..models.approval_readiness_view import ApprovalReadinessView
  from ..models.approval_route_view import ApprovalRouteView
  from ..models.approval_rule_match import ApprovalRuleMatch
  from ..models.approval_schedule_view import ApprovalScheduleView
  from ..models.approval_sms_snapshot import ApprovalSMSSnapshot
  from ..models.approval_snapshot_summary import ApprovalSnapshotSummary
  from ..models.approval_transactional_snapshot import ApprovalTransactionalSnapshot
  from ..models.approval_website_snapshot import ApprovalWebsiteSnapshot
  from ..models.approval_whats_app_snapshot import ApprovalWhatsAppSnapshot
  from ..models.review_held import ReviewHeld





T = TypeVar("T", bound="ApprovalDetail")



@_attrs_define
class ApprovalDetail:
    """ 
        Attributes:
            approvers (list[ApprovalPerson]):
            author (ApprovalPerson):
            can (ApprovalCan):
            changes (list[ApprovalChange]):
            changes_since (list[ApprovalChange]):
            channel (ApprovalDetailChannel):
            comments (list[ApprovalCommentView]):
            decisions (list[ApprovalDecisionView]):
            given (int):
            id (str):
            name (str):
            needed (int):
            note (str):
            requested_at (datetime.datetime):
            requested_by (ApprovalPerson):
            rules (list[ApprovalRuleMatch]):
            snapshot (ApprovalSnapshotSummary):
            status (ApprovalDetailStatus):
            subject_type (ApprovalDetailSubjectType):
            version (int):
            audience (ApprovalAudienceView | Unset):
            audit (ApprovalAuditLine | Unset):
            automation (ApprovalAutomationSnapshot | Unset):
            bypass_reason (str | Unset):
            campaign (ApprovalCampaignView | Unset):
            client_signoff (ApprovalClientSignoffState | Unset):
            comment (str | Unset):
            decided_at (datetime.datetime | Unset):
            decided_by (str | Unset):
            follows_id (str | Unset):
            held (ReviewHeld | Unset):
            invalidated_at (datetime.datetime | Unset):
            invalidated_by (str | Unset):
            invalidated_fields (str | Unset):
            invalidated_reason (str | Unset):
            next_id (str | Unset):
            page (ApprovalPageSnapshot | Unset):
            push (ApprovalPushSnapshot | Unset):
            readiness (ApprovalReadinessView | Unset):
            route (ApprovalRouteView | Unset):
            schedule (ApprovalScheduleView | Unset):
            sms (ApprovalSMSSnapshot | Unset):
            transactional (ApprovalTransactionalSnapshot | Unset):
            website (ApprovalWebsiteSnapshot | Unset):
            whatsapp (ApprovalWhatsAppSnapshot | Unset):
     """

    approvers: list[ApprovalPerson]
    author: ApprovalPerson
    can: ApprovalCan
    changes: list[ApprovalChange]
    changes_since: list[ApprovalChange]
    channel: ApprovalDetailChannel
    comments: list[ApprovalCommentView]
    decisions: list[ApprovalDecisionView]
    given: int
    id: str
    name: str
    needed: int
    note: str
    requested_at: datetime.datetime
    requested_by: ApprovalPerson
    rules: list[ApprovalRuleMatch]
    snapshot: ApprovalSnapshotSummary
    status: ApprovalDetailStatus
    subject_type: ApprovalDetailSubjectType
    version: int
    audience: ApprovalAudienceView | Unset = UNSET
    audit: ApprovalAuditLine | Unset = UNSET
    automation: ApprovalAutomationSnapshot | Unset = UNSET
    bypass_reason: str | Unset = UNSET
    campaign: ApprovalCampaignView | Unset = UNSET
    client_signoff: ApprovalClientSignoffState | Unset = UNSET
    comment: str | Unset = UNSET
    decided_at: datetime.datetime | Unset = UNSET
    decided_by: str | Unset = UNSET
    follows_id: str | Unset = UNSET
    held: ReviewHeld | Unset = UNSET
    invalidated_at: datetime.datetime | Unset = UNSET
    invalidated_by: str | Unset = UNSET
    invalidated_fields: str | Unset = UNSET
    invalidated_reason: str | Unset = UNSET
    next_id: str | Unset = UNSET
    page: ApprovalPageSnapshot | Unset = UNSET
    push: ApprovalPushSnapshot | Unset = UNSET
    readiness: ApprovalReadinessView | Unset = UNSET
    route: ApprovalRouteView | Unset = UNSET
    schedule: ApprovalScheduleView | Unset = UNSET
    sms: ApprovalSMSSnapshot | Unset = UNSET
    transactional: ApprovalTransactionalSnapshot | Unset = UNSET
    website: ApprovalWebsiteSnapshot | Unset = UNSET
    whatsapp: ApprovalWhatsAppSnapshot | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.approval_audience_view import ApprovalAudienceView # noqa: PLC0415
        from ..models.approval_audit_line import ApprovalAuditLine # noqa: PLC0415
        from ..models.approval_automation_snapshot import ApprovalAutomationSnapshot # noqa: PLC0415
        from ..models.approval_campaign_view import ApprovalCampaignView # noqa: PLC0415
        from ..models.approval_can import ApprovalCan # noqa: PLC0415
        from ..models.approval_change import ApprovalChange # noqa: PLC0415
        from ..models.approval_client_signoff_state import ApprovalClientSignoffState # noqa: PLC0415
        from ..models.approval_comment_view import ApprovalCommentView # noqa: PLC0415
        from ..models.approval_decision_view import ApprovalDecisionView # noqa: PLC0415
        from ..models.approval_page_snapshot import ApprovalPageSnapshot # noqa: PLC0415
        from ..models.approval_person import ApprovalPerson # noqa: PLC0415
        from ..models.approval_push_snapshot import ApprovalPushSnapshot # noqa: PLC0415
        from ..models.approval_readiness_view import ApprovalReadinessView # noqa: PLC0415
        from ..models.approval_route_view import ApprovalRouteView # noqa: PLC0415
        from ..models.approval_rule_match import ApprovalRuleMatch # noqa: PLC0415
        from ..models.approval_schedule_view import ApprovalScheduleView # noqa: PLC0415
        from ..models.approval_sms_snapshot import ApprovalSMSSnapshot # noqa: PLC0415
        from ..models.approval_snapshot_summary import ApprovalSnapshotSummary # noqa: PLC0415
        from ..models.approval_transactional_snapshot import ApprovalTransactionalSnapshot # noqa: PLC0415
        from ..models.approval_website_snapshot import ApprovalWebsiteSnapshot # noqa: PLC0415
        from ..models.approval_whats_app_snapshot import ApprovalWhatsAppSnapshot # noqa: PLC0415
        from ..models.review_held import ReviewHeld # noqa: PLC0415
        approvers = []
        for approvers_item_data in self.approvers:
            approvers_item = approvers_item_data.to_dict()
            approvers.append(approvers_item)



        author = self.author.to_dict()

        can = self.can.to_dict()

        changes = []
        for changes_item_data in self.changes:
            changes_item = changes_item_data.to_dict()
            changes.append(changes_item)



        changes_since = []
        for changes_since_item_data in self.changes_since:
            changes_since_item = changes_since_item_data.to_dict()
            changes_since.append(changes_since_item)



        channel = self.channel.value

        comments = []
        for comments_item_data in self.comments:
            comments_item = comments_item_data.to_dict()
            comments.append(comments_item)



        decisions = []
        for decisions_item_data in self.decisions:
            decisions_item = decisions_item_data.to_dict()
            decisions.append(decisions_item)



        given = self.given

        id = self.id

        name = self.name

        needed = self.needed

        note = self.note

        requested_at = self.requested_at.isoformat()

        requested_by = self.requested_by.to_dict()

        rules = []
        for rules_item_data in self.rules:
            rules_item = rules_item_data.to_dict()
            rules.append(rules_item)



        snapshot = self.snapshot.to_dict()

        status = self.status.value

        subject_type = self.subject_type.value

        version = self.version

        audience: dict[str, Any] | Unset = UNSET
        if not isinstance(self.audience, Unset):
            audience = self.audience.to_dict()

        audit: dict[str, Any] | Unset = UNSET
        if not isinstance(self.audit, Unset):
            audit = self.audit.to_dict()

        automation: dict[str, Any] | Unset = UNSET
        if not isinstance(self.automation, Unset):
            automation = self.automation.to_dict()

        bypass_reason = self.bypass_reason

        campaign: dict[str, Any] | Unset = UNSET
        if not isinstance(self.campaign, Unset):
            campaign = self.campaign.to_dict()

        client_signoff: dict[str, Any] | Unset = UNSET
        if not isinstance(self.client_signoff, Unset):
            client_signoff = self.client_signoff.to_dict()

        comment = self.comment

        decided_at: str | Unset = UNSET
        if not isinstance(self.decided_at, Unset):
            decided_at = self.decided_at.isoformat()

        decided_by = self.decided_by

        follows_id = self.follows_id

        held: dict[str, Any] | Unset = UNSET
        if not isinstance(self.held, Unset):
            held = self.held.to_dict()

        invalidated_at: str | Unset = UNSET
        if not isinstance(self.invalidated_at, Unset):
            invalidated_at = self.invalidated_at.isoformat()

        invalidated_by = self.invalidated_by

        invalidated_fields = self.invalidated_fields

        invalidated_reason = self.invalidated_reason

        next_id = self.next_id

        page: dict[str, Any] | Unset = UNSET
        if not isinstance(self.page, Unset):
            page = self.page.to_dict()

        push: dict[str, Any] | Unset = UNSET
        if not isinstance(self.push, Unset):
            push = self.push.to_dict()

        readiness: dict[str, Any] | Unset = UNSET
        if not isinstance(self.readiness, Unset):
            readiness = self.readiness.to_dict()

        route: dict[str, Any] | Unset = UNSET
        if not isinstance(self.route, Unset):
            route = self.route.to_dict()

        schedule: dict[str, Any] | Unset = UNSET
        if not isinstance(self.schedule, Unset):
            schedule = self.schedule.to_dict()

        sms: dict[str, Any] | Unset = UNSET
        if not isinstance(self.sms, Unset):
            sms = self.sms.to_dict()

        transactional: dict[str, Any] | Unset = UNSET
        if not isinstance(self.transactional, Unset):
            transactional = self.transactional.to_dict()

        website: dict[str, Any] | Unset = UNSET
        if not isinstance(self.website, Unset):
            website = self.website.to_dict()

        whatsapp: dict[str, Any] | Unset = UNSET
        if not isinstance(self.whatsapp, Unset):
            whatsapp = self.whatsapp.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "approvers": approvers,
            "author": author,
            "can": can,
            "changes": changes,
            "changesSince": changes_since,
            "channel": channel,
            "comments": comments,
            "decisions": decisions,
            "given": given,
            "id": id,
            "name": name,
            "needed": needed,
            "note": note,
            "requestedAt": requested_at,
            "requestedBy": requested_by,
            "rules": rules,
            "snapshot": snapshot,
            "status": status,
            "subjectType": subject_type,
            "version": version,
        })
        if audience is not UNSET:
            field_dict["audience"] = audience
        if audit is not UNSET:
            field_dict["audit"] = audit
        if automation is not UNSET:
            field_dict["automation"] = automation
        if bypass_reason is not UNSET:
            field_dict["bypassReason"] = bypass_reason
        if campaign is not UNSET:
            field_dict["campaign"] = campaign
        if client_signoff is not UNSET:
            field_dict["clientSignoff"] = client_signoff
        if comment is not UNSET:
            field_dict["comment"] = comment
        if decided_at is not UNSET:
            field_dict["decidedAt"] = decided_at
        if decided_by is not UNSET:
            field_dict["decidedBy"] = decided_by
        if follows_id is not UNSET:
            field_dict["followsId"] = follows_id
        if held is not UNSET:
            field_dict["held"] = held
        if invalidated_at is not UNSET:
            field_dict["invalidatedAt"] = invalidated_at
        if invalidated_by is not UNSET:
            field_dict["invalidatedBy"] = invalidated_by
        if invalidated_fields is not UNSET:
            field_dict["invalidatedFields"] = invalidated_fields
        if invalidated_reason is not UNSET:
            field_dict["invalidatedReason"] = invalidated_reason
        if next_id is not UNSET:
            field_dict["nextId"] = next_id
        if page is not UNSET:
            field_dict["page"] = page
        if push is not UNSET:
            field_dict["push"] = push
        if readiness is not UNSET:
            field_dict["readiness"] = readiness
        if route is not UNSET:
            field_dict["route"] = route
        if schedule is not UNSET:
            field_dict["schedule"] = schedule
        if sms is not UNSET:
            field_dict["sms"] = sms
        if transactional is not UNSET:
            field_dict["transactional"] = transactional
        if website is not UNSET:
            field_dict["website"] = website
        if whatsapp is not UNSET:
            field_dict["whatsapp"] = whatsapp

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.approval_audience_view import ApprovalAudienceView # noqa: PLC0415
        from ..models.approval_audit_line import ApprovalAuditLine # noqa: PLC0415
        from ..models.approval_automation_snapshot import ApprovalAutomationSnapshot # noqa: PLC0415
        from ..models.approval_campaign_view import ApprovalCampaignView # noqa: PLC0415
        from ..models.approval_can import ApprovalCan # noqa: PLC0415
        from ..models.approval_change import ApprovalChange # noqa: PLC0415
        from ..models.approval_client_signoff_state import ApprovalClientSignoffState # noqa: PLC0415
        from ..models.approval_comment_view import ApprovalCommentView # noqa: PLC0415
        from ..models.approval_decision_view import ApprovalDecisionView # noqa: PLC0415
        from ..models.approval_page_snapshot import ApprovalPageSnapshot # noqa: PLC0415
        from ..models.approval_person import ApprovalPerson # noqa: PLC0415
        from ..models.approval_push_snapshot import ApprovalPushSnapshot # noqa: PLC0415
        from ..models.approval_readiness_view import ApprovalReadinessView # noqa: PLC0415
        from ..models.approval_route_view import ApprovalRouteView # noqa: PLC0415
        from ..models.approval_rule_match import ApprovalRuleMatch # noqa: PLC0415
        from ..models.approval_schedule_view import ApprovalScheduleView # noqa: PLC0415
        from ..models.approval_sms_snapshot import ApprovalSMSSnapshot # noqa: PLC0415
        from ..models.approval_snapshot_summary import ApprovalSnapshotSummary # noqa: PLC0415
        from ..models.approval_transactional_snapshot import ApprovalTransactionalSnapshot # noqa: PLC0415
        from ..models.approval_website_snapshot import ApprovalWebsiteSnapshot # noqa: PLC0415
        from ..models.approval_whats_app_snapshot import ApprovalWhatsAppSnapshot # noqa: PLC0415
        from ..models.review_held import ReviewHeld # noqa: PLC0415
        d = dict(src_dict)
        approvers = []
        _approvers = d.pop("approvers")
        for approvers_item_data in (_approvers):
            approvers_item = ApprovalPerson.from_dict(approvers_item_data)



            approvers.append(approvers_item)


        author = ApprovalPerson.from_dict(d.pop("author"))




        can = ApprovalCan.from_dict(d.pop("can"))




        changes = []
        _changes = d.pop("changes")
        for changes_item_data in (_changes):
            changes_item = ApprovalChange.from_dict(changes_item_data)



            changes.append(changes_item)


        changes_since = []
        _changes_since = d.pop("changesSince")
        for changes_since_item_data in (_changes_since):
            changes_since_item = ApprovalChange.from_dict(changes_since_item_data)



            changes_since.append(changes_since_item)


        channel = ApprovalDetailChannel(d.pop("channel"))




        comments = []
        _comments = d.pop("comments")
        for comments_item_data in (_comments):
            comments_item = ApprovalCommentView.from_dict(comments_item_data)



            comments.append(comments_item)


        decisions = []
        _decisions = d.pop("decisions")
        for decisions_item_data in (_decisions):
            decisions_item = ApprovalDecisionView.from_dict(decisions_item_data)



            decisions.append(decisions_item)


        given = d.pop("given")

        id = d.pop("id")

        name = d.pop("name")

        needed = d.pop("needed")

        note = d.pop("note")

        requested_at = datetime.datetime.fromisoformat(d.pop("requestedAt"))




        requested_by = ApprovalPerson.from_dict(d.pop("requestedBy"))




        rules = []
        _rules = d.pop("rules")
        for rules_item_data in (_rules):
            rules_item = ApprovalRuleMatch.from_dict(rules_item_data)



            rules.append(rules_item)


        snapshot = ApprovalSnapshotSummary.from_dict(d.pop("snapshot"))




        status = ApprovalDetailStatus(d.pop("status"))




        subject_type = ApprovalDetailSubjectType(d.pop("subjectType"))




        version = d.pop("version")

        _audience = d.pop("audience", UNSET)
        audience: ApprovalAudienceView | Unset
        if isinstance(_audience,  Unset):
            audience = UNSET
        else:
            audience = ApprovalAudienceView.from_dict(_audience)




        _audit = d.pop("audit", UNSET)
        audit: ApprovalAuditLine | Unset
        if isinstance(_audit,  Unset):
            audit = UNSET
        else:
            audit = ApprovalAuditLine.from_dict(_audit)




        _automation = d.pop("automation", UNSET)
        automation: ApprovalAutomationSnapshot | Unset
        if isinstance(_automation,  Unset):
            automation = UNSET
        else:
            automation = ApprovalAutomationSnapshot.from_dict(_automation)




        bypass_reason = d.pop("bypassReason", UNSET)

        _campaign = d.pop("campaign", UNSET)
        campaign: ApprovalCampaignView | Unset
        if isinstance(_campaign,  Unset):
            campaign = UNSET
        else:
            campaign = ApprovalCampaignView.from_dict(_campaign)




        _client_signoff = d.pop("clientSignoff", UNSET)
        client_signoff: ApprovalClientSignoffState | Unset
        if isinstance(_client_signoff,  Unset):
            client_signoff = UNSET
        else:
            client_signoff = ApprovalClientSignoffState.from_dict(_client_signoff)




        comment = d.pop("comment", UNSET)

        _decided_at = d.pop("decidedAt", UNSET)
        decided_at: datetime.datetime | Unset
        if isinstance(_decided_at,  Unset):
            decided_at = UNSET
        else:
            decided_at = datetime.datetime.fromisoformat(_decided_at)




        decided_by = d.pop("decidedBy", UNSET)

        follows_id = d.pop("followsId", UNSET)

        _held = d.pop("held", UNSET)
        held: ReviewHeld | Unset
        if isinstance(_held,  Unset):
            held = UNSET
        else:
            held = ReviewHeld.from_dict(_held)




        _invalidated_at = d.pop("invalidatedAt", UNSET)
        invalidated_at: datetime.datetime | Unset
        if isinstance(_invalidated_at,  Unset):
            invalidated_at = UNSET
        else:
            invalidated_at = datetime.datetime.fromisoformat(_invalidated_at)




        invalidated_by = d.pop("invalidatedBy", UNSET)

        invalidated_fields = d.pop("invalidatedFields", UNSET)

        invalidated_reason = d.pop("invalidatedReason", UNSET)

        next_id = d.pop("nextId", UNSET)

        _page = d.pop("page", UNSET)
        page: ApprovalPageSnapshot | Unset
        if isinstance(_page,  Unset):
            page = UNSET
        else:
            page = ApprovalPageSnapshot.from_dict(_page)




        _push = d.pop("push", UNSET)
        push: ApprovalPushSnapshot | Unset
        if isinstance(_push,  Unset):
            push = UNSET
        else:
            push = ApprovalPushSnapshot.from_dict(_push)




        _readiness = d.pop("readiness", UNSET)
        readiness: ApprovalReadinessView | Unset
        if isinstance(_readiness,  Unset):
            readiness = UNSET
        else:
            readiness = ApprovalReadinessView.from_dict(_readiness)




        _route = d.pop("route", UNSET)
        route: ApprovalRouteView | Unset
        if isinstance(_route,  Unset):
            route = UNSET
        else:
            route = ApprovalRouteView.from_dict(_route)




        _schedule = d.pop("schedule", UNSET)
        schedule: ApprovalScheduleView | Unset
        if isinstance(_schedule,  Unset):
            schedule = UNSET
        else:
            schedule = ApprovalScheduleView.from_dict(_schedule)




        _sms = d.pop("sms", UNSET)
        sms: ApprovalSMSSnapshot | Unset
        if isinstance(_sms,  Unset):
            sms = UNSET
        else:
            sms = ApprovalSMSSnapshot.from_dict(_sms)




        _transactional = d.pop("transactional", UNSET)
        transactional: ApprovalTransactionalSnapshot | Unset
        if isinstance(_transactional,  Unset):
            transactional = UNSET
        else:
            transactional = ApprovalTransactionalSnapshot.from_dict(_transactional)




        _website = d.pop("website", UNSET)
        website: ApprovalWebsiteSnapshot | Unset
        if isinstance(_website,  Unset):
            website = UNSET
        else:
            website = ApprovalWebsiteSnapshot.from_dict(_website)




        _whatsapp = d.pop("whatsapp", UNSET)
        whatsapp: ApprovalWhatsAppSnapshot | Unset
        if isinstance(_whatsapp,  Unset):
            whatsapp = UNSET
        else:
            whatsapp = ApprovalWhatsAppSnapshot.from_dict(_whatsapp)




        approval_detail = cls(
            approvers=approvers,
            author=author,
            can=can,
            changes=changes,
            changes_since=changes_since,
            channel=channel,
            comments=comments,
            decisions=decisions,
            given=given,
            id=id,
            name=name,
            needed=needed,
            note=note,
            requested_at=requested_at,
            requested_by=requested_by,
            rules=rules,
            snapshot=snapshot,
            status=status,
            subject_type=subject_type,
            version=version,
            audience=audience,
            audit=audit,
            automation=automation,
            bypass_reason=bypass_reason,
            campaign=campaign,
            client_signoff=client_signoff,
            comment=comment,
            decided_at=decided_at,
            decided_by=decided_by,
            follows_id=follows_id,
            held=held,
            invalidated_at=invalidated_at,
            invalidated_by=invalidated_by,
            invalidated_fields=invalidated_fields,
            invalidated_reason=invalidated_reason,
            next_id=next_id,
            page=page,
            push=push,
            readiness=readiness,
            route=route,
            schedule=schedule,
            sms=sms,
            transactional=transactional,
            website=website,
            whatsapp=whatsapp,
        )

        return approval_detail

