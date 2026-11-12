from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="CardIssue")



@_attrs_define
class CardIssue:
    """ 
        Attributes:
            booking_id (str):
            issue (int):
            ready (bool):
            sponsor (str):
            creative_due (datetime.datetime | Unset):
     """

    booking_id: str
    issue: int
    ready: bool
    sponsor: str
    creative_due: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        booking_id = self.booking_id

        issue = self.issue

        ready = self.ready

        sponsor = self.sponsor

        creative_due: str | Unset = UNSET
        if not isinstance(self.creative_due, Unset):
            creative_due = self.creative_due.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "bookingId": booking_id,
            "issue": issue,
            "ready": ready,
            "sponsor": sponsor,
        })
        if creative_due is not UNSET:
            field_dict["creativeDue"] = creative_due

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        booking_id = d.pop("bookingId")

        issue = d.pop("issue")

        ready = d.pop("ready")

        sponsor = d.pop("sponsor")

        _creative_due = d.pop("creativeDue", UNSET)
        creative_due: datetime.datetime | Unset
        if isinstance(_creative_due,  Unset):
            creative_due = UNSET
        else:
            creative_due = datetime.datetime.fromisoformat(_creative_due)




        card_issue = cls(
            booking_id=booking_id,
            issue=issue,
            ready=ready,
            sponsor=sponsor,
            creative_due=creative_due,
        )

        return card_issue

