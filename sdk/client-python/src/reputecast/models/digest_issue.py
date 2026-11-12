from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.digest_issue_hold_reason import DigestIssueHoldReason
from ..models.digest_issue_skip_reason import DigestIssueSkipReason
from ..models.digest_issue_status import DigestIssueStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.digest_feed_try import DigestFeedTry
  from ..models.digest_issue_approval import DigestIssueApproval





T = TypeVar("T", bound="DigestIssue")



@_attrs_define
class DigestIssue:
    """ 
        Attributes:
            estimated (bool):
            feed_tries (list[DigestFeedTry]):
            first_issue (bool):
            id (str):
            needs_approval (bool):
            posts (int):
            status (DigestIssueStatus):
            approval (DigestIssueApproval | Unset):
            campaign_id (str | Unset):
            clicks (float | Unset):
            clicks_delta (float | Unset):
            deadline (datetime.datetime | Unset):
            delivered (float | Unset):
            hold_reason (DigestIssueHoldReason | Unset):
            human_opens (float | Unset):
            human_opens_delta (float | Unset):
            number (int | Unset):
            posts_delta (int | Unset):
            prepared_at (datetime.datetime | Unset):
            recipients (int | Unset):
            recipients_delta (int | Unset):
            send_at (datetime.datetime | Unset):
            sent_at (datetime.datetime | Unset):
            skip_note (str | Unset):
            skip_reason (DigestIssueSkipReason | Unset):
            skipped_at (datetime.datetime | Unset):
            skipped_by (str | Unset):
     """

    estimated: bool
    feed_tries: list[DigestFeedTry]
    first_issue: bool
    id: str
    needs_approval: bool
    posts: int
    status: DigestIssueStatus
    approval: DigestIssueApproval | Unset = UNSET
    campaign_id: str | Unset = UNSET
    clicks: float | Unset = UNSET
    clicks_delta: float | Unset = UNSET
    deadline: datetime.datetime | Unset = UNSET
    delivered: float | Unset = UNSET
    hold_reason: DigestIssueHoldReason | Unset = UNSET
    human_opens: float | Unset = UNSET
    human_opens_delta: float | Unset = UNSET
    number: int | Unset = UNSET
    posts_delta: int | Unset = UNSET
    prepared_at: datetime.datetime | Unset = UNSET
    recipients: int | Unset = UNSET
    recipients_delta: int | Unset = UNSET
    send_at: datetime.datetime | Unset = UNSET
    sent_at: datetime.datetime | Unset = UNSET
    skip_note: str | Unset = UNSET
    skip_reason: DigestIssueSkipReason | Unset = UNSET
    skipped_at: datetime.datetime | Unset = UNSET
    skipped_by: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.digest_feed_try import DigestFeedTry # noqa: PLC0415
        from ..models.digest_issue_approval import DigestIssueApproval # noqa: PLC0415
        estimated = self.estimated

        feed_tries = []
        for feed_tries_item_data in self.feed_tries:
            feed_tries_item = feed_tries_item_data.to_dict()
            feed_tries.append(feed_tries_item)



        first_issue = self.first_issue

        id = self.id

        needs_approval = self.needs_approval

        posts = self.posts

        status = self.status.value

        approval: dict[str, Any] | Unset = UNSET
        if not isinstance(self.approval, Unset):
            approval = self.approval.to_dict()

        campaign_id = self.campaign_id

        clicks = self.clicks

        clicks_delta = self.clicks_delta

        deadline: str | Unset = UNSET
        if not isinstance(self.deadline, Unset):
            deadline = self.deadline.isoformat()

        delivered = self.delivered

        hold_reason: str | Unset = UNSET
        if not isinstance(self.hold_reason, Unset):
            hold_reason = self.hold_reason.value


        human_opens = self.human_opens

        human_opens_delta = self.human_opens_delta

        number = self.number

        posts_delta = self.posts_delta

        prepared_at: str | Unset = UNSET
        if not isinstance(self.prepared_at, Unset):
            prepared_at = self.prepared_at.isoformat()

        recipients = self.recipients

        recipients_delta = self.recipients_delta

        send_at: str | Unset = UNSET
        if not isinstance(self.send_at, Unset):
            send_at = self.send_at.isoformat()

        sent_at: str | Unset = UNSET
        if not isinstance(self.sent_at, Unset):
            sent_at = self.sent_at.isoformat()

        skip_note = self.skip_note

        skip_reason: str | Unset = UNSET
        if not isinstance(self.skip_reason, Unset):
            skip_reason = self.skip_reason.value


        skipped_at: str | Unset = UNSET
        if not isinstance(self.skipped_at, Unset):
            skipped_at = self.skipped_at.isoformat()

        skipped_by = self.skipped_by


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "estimated": estimated,
            "feedTries": feed_tries,
            "firstIssue": first_issue,
            "id": id,
            "needsApproval": needs_approval,
            "posts": posts,
            "status": status,
        })
        if approval is not UNSET:
            field_dict["approval"] = approval
        if campaign_id is not UNSET:
            field_dict["campaignId"] = campaign_id
        if clicks is not UNSET:
            field_dict["clicks"] = clicks
        if clicks_delta is not UNSET:
            field_dict["clicksDelta"] = clicks_delta
        if deadline is not UNSET:
            field_dict["deadline"] = deadline
        if delivered is not UNSET:
            field_dict["delivered"] = delivered
        if hold_reason is not UNSET:
            field_dict["holdReason"] = hold_reason
        if human_opens is not UNSET:
            field_dict["humanOpens"] = human_opens
        if human_opens_delta is not UNSET:
            field_dict["humanOpensDelta"] = human_opens_delta
        if number is not UNSET:
            field_dict["number"] = number
        if posts_delta is not UNSET:
            field_dict["postsDelta"] = posts_delta
        if prepared_at is not UNSET:
            field_dict["preparedAt"] = prepared_at
        if recipients is not UNSET:
            field_dict["recipients"] = recipients
        if recipients_delta is not UNSET:
            field_dict["recipientsDelta"] = recipients_delta
        if send_at is not UNSET:
            field_dict["sendAt"] = send_at
        if sent_at is not UNSET:
            field_dict["sentAt"] = sent_at
        if skip_note is not UNSET:
            field_dict["skipNote"] = skip_note
        if skip_reason is not UNSET:
            field_dict["skipReason"] = skip_reason
        if skipped_at is not UNSET:
            field_dict["skippedAt"] = skipped_at
        if skipped_by is not UNSET:
            field_dict["skippedBy"] = skipped_by

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.digest_feed_try import DigestFeedTry # noqa: PLC0415
        from ..models.digest_issue_approval import DigestIssueApproval # noqa: PLC0415
        d = dict(src_dict)
        estimated = d.pop("estimated")

        feed_tries = []
        _feed_tries = d.pop("feedTries")
        for feed_tries_item_data in (_feed_tries):
            feed_tries_item = DigestFeedTry.from_dict(feed_tries_item_data)



            feed_tries.append(feed_tries_item)


        first_issue = d.pop("firstIssue")

        id = d.pop("id")

        needs_approval = d.pop("needsApproval")

        posts = d.pop("posts")

        status = DigestIssueStatus(d.pop("status"))




        _approval = d.pop("approval", UNSET)
        approval: DigestIssueApproval | Unset
        if isinstance(_approval,  Unset):
            approval = UNSET
        else:
            approval = DigestIssueApproval.from_dict(_approval)




        campaign_id = d.pop("campaignId", UNSET)

        clicks = d.pop("clicks", UNSET)

        clicks_delta = d.pop("clicksDelta", UNSET)

        _deadline = d.pop("deadline", UNSET)
        deadline: datetime.datetime | Unset
        if isinstance(_deadline,  Unset):
            deadline = UNSET
        else:
            deadline = datetime.datetime.fromisoformat(_deadline)




        delivered = d.pop("delivered", UNSET)

        _hold_reason = d.pop("holdReason", UNSET)
        hold_reason: DigestIssueHoldReason | Unset
        if isinstance(_hold_reason,  Unset):
            hold_reason = UNSET
        else:
            hold_reason = DigestIssueHoldReason(_hold_reason)




        human_opens = d.pop("humanOpens", UNSET)

        human_opens_delta = d.pop("humanOpensDelta", UNSET)

        number = d.pop("number", UNSET)

        posts_delta = d.pop("postsDelta", UNSET)

        _prepared_at = d.pop("preparedAt", UNSET)
        prepared_at: datetime.datetime | Unset
        if isinstance(_prepared_at,  Unset):
            prepared_at = UNSET
        else:
            prepared_at = datetime.datetime.fromisoformat(_prepared_at)




        recipients = d.pop("recipients", UNSET)

        recipients_delta = d.pop("recipientsDelta", UNSET)

        _send_at = d.pop("sendAt", UNSET)
        send_at: datetime.datetime | Unset
        if isinstance(_send_at,  Unset):
            send_at = UNSET
        else:
            send_at = datetime.datetime.fromisoformat(_send_at)




        _sent_at = d.pop("sentAt", UNSET)
        sent_at: datetime.datetime | Unset
        if isinstance(_sent_at,  Unset):
            sent_at = UNSET
        else:
            sent_at = datetime.datetime.fromisoformat(_sent_at)




        skip_note = d.pop("skipNote", UNSET)

        _skip_reason = d.pop("skipReason", UNSET)
        skip_reason: DigestIssueSkipReason | Unset
        if isinstance(_skip_reason,  Unset):
            skip_reason = UNSET
        else:
            skip_reason = DigestIssueSkipReason(_skip_reason)




        _skipped_at = d.pop("skippedAt", UNSET)
        skipped_at: datetime.datetime | Unset
        if isinstance(_skipped_at,  Unset):
            skipped_at = UNSET
        else:
            skipped_at = datetime.datetime.fromisoformat(_skipped_at)




        skipped_by = d.pop("skippedBy", UNSET)

        digest_issue = cls(
            estimated=estimated,
            feed_tries=feed_tries,
            first_issue=first_issue,
            id=id,
            needs_approval=needs_approval,
            posts=posts,
            status=status,
            approval=approval,
            campaign_id=campaign_id,
            clicks=clicks,
            clicks_delta=clicks_delta,
            deadline=deadline,
            delivered=delivered,
            hold_reason=hold_reason,
            human_opens=human_opens,
            human_opens_delta=human_opens_delta,
            number=number,
            posts_delta=posts_delta,
            prepared_at=prepared_at,
            recipients=recipients,
            recipients_delta=recipients_delta,
            send_at=send_at,
            sent_at=sent_at,
            skip_note=skip_note,
            skip_reason=skip_reason,
            skipped_at=skipped_at,
            skipped_by=skipped_by,
        )

        return digest_issue

