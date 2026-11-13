from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="SponsorRequestIssue")



@_attrs_define
class SponsorRequestIssue:
    """ 
        Attributes:
            issue_on (datetime.datetime):
            number (int):
            payment_due_at (datetime.datetime):
            taken (bool):
            held_by (str | Unset):
            held_until (datetime.datetime | Unset):
     """

    issue_on: datetime.datetime
    number: int
    payment_due_at: datetime.datetime
    taken: bool
    held_by: str | Unset = UNSET
    held_until: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        issue_on = self.issue_on.isoformat()

        number = self.number

        payment_due_at = self.payment_due_at.isoformat()

        taken = self.taken

        held_by = self.held_by

        held_until: str | Unset = UNSET
        if not isinstance(self.held_until, Unset):
            held_until = self.held_until.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "issueOn": issue_on,
            "number": number,
            "paymentDueAt": payment_due_at,
            "taken": taken,
        })
        if held_by is not UNSET:
            field_dict["heldBy"] = held_by
        if held_until is not UNSET:
            field_dict["heldUntil"] = held_until

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        issue_on = datetime.datetime.fromisoformat(d.pop("issueOn"))




        number = d.pop("number")

        payment_due_at = datetime.datetime.fromisoformat(d.pop("paymentDueAt"))




        taken = d.pop("taken")

        held_by = d.pop("heldBy", UNSET)

        _held_until = d.pop("heldUntil", UNSET)
        held_until: datetime.datetime | Unset
        if isinstance(_held_until,  Unset):
            held_until = UNSET
        else:
            held_until = datetime.datetime.fromisoformat(_held_until)




        sponsor_request_issue = cls(
            issue_on=issue_on,
            number=number,
            payment_due_at=payment_due_at,
            taken=taken,
            held_by=held_by,
            held_until=held_until,
        )

        return sponsor_request_issue

