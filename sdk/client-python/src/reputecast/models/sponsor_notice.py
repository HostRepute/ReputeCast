from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.sponsor_notice_kind import SponsorNoticeKind
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="SponsorNotice")



@_attrs_define
class SponsorNotice:
    """ 
        Attributes:
            amount_cents (int):
            booking_id (str):
            contact_email (str):
            contact_name (str):
            days_to_issue (int):
            issue (int):
            issue_on (datetime.datetime):
            kind (SponsorNoticeKind):
            list_name (str):
            slot_id (str):
            sponsor (str):
            due (datetime.datetime | Unset):
     """

    amount_cents: int
    booking_id: str
    contact_email: str
    contact_name: str
    days_to_issue: int
    issue: int
    issue_on: datetime.datetime
    kind: SponsorNoticeKind
    list_name: str
    slot_id: str
    sponsor: str
    due: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        amount_cents = self.amount_cents

        booking_id = self.booking_id

        contact_email = self.contact_email

        contact_name = self.contact_name

        days_to_issue = self.days_to_issue

        issue = self.issue

        issue_on = self.issue_on.isoformat()

        kind = self.kind.value

        list_name = self.list_name

        slot_id = self.slot_id

        sponsor = self.sponsor

        due: str | Unset = UNSET
        if not isinstance(self.due, Unset):
            due = self.due.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "amountCents": amount_cents,
            "bookingId": booking_id,
            "contactEmail": contact_email,
            "contactName": contact_name,
            "daysToIssue": days_to_issue,
            "issue": issue,
            "issueOn": issue_on,
            "kind": kind,
            "listName": list_name,
            "slotId": slot_id,
            "sponsor": sponsor,
        })
        if due is not UNSET:
            field_dict["due"] = due

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        amount_cents = d.pop("amountCents")

        booking_id = d.pop("bookingId")

        contact_email = d.pop("contactEmail")

        contact_name = d.pop("contactName")

        days_to_issue = d.pop("daysToIssue")

        issue = d.pop("issue")

        issue_on = datetime.datetime.fromisoformat(d.pop("issueOn"))




        kind = SponsorNoticeKind(d.pop("kind"))




        list_name = d.pop("listName")

        slot_id = d.pop("slotId")

        sponsor = d.pop("sponsor")

        _due = d.pop("due", UNSET)
        due: datetime.datetime | Unset
        if isinstance(_due,  Unset):
            due = UNSET
        else:
            due = datetime.datetime.fromisoformat(_due)




        sponsor_notice = cls(
            amount_cents=amount_cents,
            booking_id=booking_id,
            contact_email=contact_email,
            contact_name=contact_name,
            days_to_issue=days_to_issue,
            issue=issue,
            issue_on=issue_on,
            kind=kind,
            list_name=list_name,
            slot_id=slot_id,
            sponsor=sponsor,
            due=due,
        )

        return sponsor_notice

