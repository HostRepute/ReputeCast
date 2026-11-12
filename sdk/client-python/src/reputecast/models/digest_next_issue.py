from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.digest_next_issue_hold_reason import DigestNextIssueHoldReason
from ..models.digest_next_issue_status import DigestNextIssueStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.digest_feed_item import DigestFeedItem
  from ..models.digest_feed_try import DigestFeedTry
  from ..models.digest_issue_approval import DigestIssueApproval
  from ..models.digest_issue_readiness import DigestIssueReadiness
  from ..models.digest_subject import DigestSubject





T = TypeVar("T", bound="DigestNextIssue")



@_attrs_define
class DigestNextIssue:
    """ 
        Attributes:
            feed_tries (list[DigestFeedTry]):
            fit (int):
            id (str):
            items (list[DigestFeedItem]):
            number (int):
            posts (int):
            preheader (str):
            recipients (int):
            status (DigestNextIssueStatus):
            subject (str):
            subjects (list[DigestSubject]):
            approval (DigestIssueApproval | Unset):
            campaign_id (str | Unset):
            deadline (datetime.datetime | Unset):
            held_at (datetime.datetime | Unset):
            hold_reason (DigestNextIssueHoldReason | Unset):
            next_try (datetime.datetime | Unset):
            prepare_at (datetime.datetime | Unset):
            prepared_at (datetime.datetime | Unset):
            readiness (DigestIssueReadiness | Unset):
            send_at (datetime.datetime | Unset):
     """

    feed_tries: list[DigestFeedTry]
    fit: int
    id: str
    items: list[DigestFeedItem]
    number: int
    posts: int
    preheader: str
    recipients: int
    status: DigestNextIssueStatus
    subject: str
    subjects: list[DigestSubject]
    approval: DigestIssueApproval | Unset = UNSET
    campaign_id: str | Unset = UNSET
    deadline: datetime.datetime | Unset = UNSET
    held_at: datetime.datetime | Unset = UNSET
    hold_reason: DigestNextIssueHoldReason | Unset = UNSET
    next_try: datetime.datetime | Unset = UNSET
    prepare_at: datetime.datetime | Unset = UNSET
    prepared_at: datetime.datetime | Unset = UNSET
    readiness: DigestIssueReadiness | Unset = UNSET
    send_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.digest_feed_item import DigestFeedItem # noqa: PLC0415
        from ..models.digest_feed_try import DigestFeedTry # noqa: PLC0415
        from ..models.digest_issue_approval import DigestIssueApproval # noqa: PLC0415
        from ..models.digest_issue_readiness import DigestIssueReadiness # noqa: PLC0415
        from ..models.digest_subject import DigestSubject # noqa: PLC0415
        feed_tries = []
        for feed_tries_item_data in self.feed_tries:
            feed_tries_item = feed_tries_item_data.to_dict()
            feed_tries.append(feed_tries_item)



        fit = self.fit

        id = self.id

        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        number = self.number

        posts = self.posts

        preheader = self.preheader

        recipients = self.recipients

        status = self.status.value

        subject = self.subject

        subjects = []
        for subjects_item_data in self.subjects:
            subjects_item = subjects_item_data.to_dict()
            subjects.append(subjects_item)



        approval: dict[str, Any] | Unset = UNSET
        if not isinstance(self.approval, Unset):
            approval = self.approval.to_dict()

        campaign_id = self.campaign_id

        deadline: str | Unset = UNSET
        if not isinstance(self.deadline, Unset):
            deadline = self.deadline.isoformat()

        held_at: str | Unset = UNSET
        if not isinstance(self.held_at, Unset):
            held_at = self.held_at.isoformat()

        hold_reason: str | Unset = UNSET
        if not isinstance(self.hold_reason, Unset):
            hold_reason = self.hold_reason.value


        next_try: str | Unset = UNSET
        if not isinstance(self.next_try, Unset):
            next_try = self.next_try.isoformat()

        prepare_at: str | Unset = UNSET
        if not isinstance(self.prepare_at, Unset):
            prepare_at = self.prepare_at.isoformat()

        prepared_at: str | Unset = UNSET
        if not isinstance(self.prepared_at, Unset):
            prepared_at = self.prepared_at.isoformat()

        readiness: dict[str, Any] | Unset = UNSET
        if not isinstance(self.readiness, Unset):
            readiness = self.readiness.to_dict()

        send_at: str | Unset = UNSET
        if not isinstance(self.send_at, Unset):
            send_at = self.send_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "feedTries": feed_tries,
            "fit": fit,
            "id": id,
            "items": items,
            "number": number,
            "posts": posts,
            "preheader": preheader,
            "recipients": recipients,
            "status": status,
            "subject": subject,
            "subjects": subjects,
        })
        if approval is not UNSET:
            field_dict["approval"] = approval
        if campaign_id is not UNSET:
            field_dict["campaignId"] = campaign_id
        if deadline is not UNSET:
            field_dict["deadline"] = deadline
        if held_at is not UNSET:
            field_dict["heldAt"] = held_at
        if hold_reason is not UNSET:
            field_dict["holdReason"] = hold_reason
        if next_try is not UNSET:
            field_dict["nextTry"] = next_try
        if prepare_at is not UNSET:
            field_dict["prepareAt"] = prepare_at
        if prepared_at is not UNSET:
            field_dict["preparedAt"] = prepared_at
        if readiness is not UNSET:
            field_dict["readiness"] = readiness
        if send_at is not UNSET:
            field_dict["sendAt"] = send_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.digest_feed_item import DigestFeedItem # noqa: PLC0415
        from ..models.digest_feed_try import DigestFeedTry # noqa: PLC0415
        from ..models.digest_issue_approval import DigestIssueApproval # noqa: PLC0415
        from ..models.digest_issue_readiness import DigestIssueReadiness # noqa: PLC0415
        from ..models.digest_subject import DigestSubject # noqa: PLC0415
        d = dict(src_dict)
        feed_tries = []
        _feed_tries = d.pop("feedTries")
        for feed_tries_item_data in (_feed_tries):
            feed_tries_item = DigestFeedTry.from_dict(feed_tries_item_data)



            feed_tries.append(feed_tries_item)


        fit = d.pop("fit")

        id = d.pop("id")

        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = DigestFeedItem.from_dict(items_item_data)



            items.append(items_item)


        number = d.pop("number")

        posts = d.pop("posts")

        preheader = d.pop("preheader")

        recipients = d.pop("recipients")

        status = DigestNextIssueStatus(d.pop("status"))




        subject = d.pop("subject")

        subjects = []
        _subjects = d.pop("subjects")
        for subjects_item_data in (_subjects):
            subjects_item = DigestSubject.from_dict(subjects_item_data)



            subjects.append(subjects_item)


        _approval = d.pop("approval", UNSET)
        approval: DigestIssueApproval | Unset
        if isinstance(_approval,  Unset):
            approval = UNSET
        else:
            approval = DigestIssueApproval.from_dict(_approval)




        campaign_id = d.pop("campaignId", UNSET)

        _deadline = d.pop("deadline", UNSET)
        deadline: datetime.datetime | Unset
        if isinstance(_deadline,  Unset):
            deadline = UNSET
        else:
            deadline = datetime.datetime.fromisoformat(_deadline)




        _held_at = d.pop("heldAt", UNSET)
        held_at: datetime.datetime | Unset
        if isinstance(_held_at,  Unset):
            held_at = UNSET
        else:
            held_at = datetime.datetime.fromisoformat(_held_at)




        _hold_reason = d.pop("holdReason", UNSET)
        hold_reason: DigestNextIssueHoldReason | Unset
        if isinstance(_hold_reason,  Unset):
            hold_reason = UNSET
        else:
            hold_reason = DigestNextIssueHoldReason(_hold_reason)




        _next_try = d.pop("nextTry", UNSET)
        next_try: datetime.datetime | Unset
        if isinstance(_next_try,  Unset):
            next_try = UNSET
        else:
            next_try = datetime.datetime.fromisoformat(_next_try)




        _prepare_at = d.pop("prepareAt", UNSET)
        prepare_at: datetime.datetime | Unset
        if isinstance(_prepare_at,  Unset):
            prepare_at = UNSET
        else:
            prepare_at = datetime.datetime.fromisoformat(_prepare_at)




        _prepared_at = d.pop("preparedAt", UNSET)
        prepared_at: datetime.datetime | Unset
        if isinstance(_prepared_at,  Unset):
            prepared_at = UNSET
        else:
            prepared_at = datetime.datetime.fromisoformat(_prepared_at)




        _readiness = d.pop("readiness", UNSET)
        readiness: DigestIssueReadiness | Unset
        if isinstance(_readiness,  Unset):
            readiness = UNSET
        else:
            readiness = DigestIssueReadiness.from_dict(_readiness)




        _send_at = d.pop("sendAt", UNSET)
        send_at: datetime.datetime | Unset
        if isinstance(_send_at,  Unset):
            send_at = UNSET
        else:
            send_at = datetime.datetime.fromisoformat(_send_at)




        digest_next_issue = cls(
            feed_tries=feed_tries,
            fit=fit,
            id=id,
            items=items,
            number=number,
            posts=posts,
            preheader=preheader,
            recipients=recipients,
            status=status,
            subject=subject,
            subjects=subjects,
            approval=approval,
            campaign_id=campaign_id,
            deadline=deadline,
            held_at=held_at,
            hold_reason=hold_reason,
            next_try=next_try,
            prepare_at=prepare_at,
            prepared_at=prepared_at,
            readiness=readiness,
            send_at=send_at,
        )

        return digest_next_issue

