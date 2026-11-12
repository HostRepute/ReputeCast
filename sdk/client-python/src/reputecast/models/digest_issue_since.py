from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="DigestIssueSince")



@_attrs_define
class DigestIssueSince:
    """ 
        Attributes:
            changed (list[str]):
            number (int):
            approved_at (datetime.datetime | Unset):
            sent_at (datetime.datetime | Unset):
     """

    changed: list[str]
    number: int
    approved_at: datetime.datetime | Unset = UNSET
    sent_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        changed = self.changed



        number = self.number

        approved_at: str | Unset = UNSET
        if not isinstance(self.approved_at, Unset):
            approved_at = self.approved_at.isoformat()

        sent_at: str | Unset = UNSET
        if not isinstance(self.sent_at, Unset):
            sent_at = self.sent_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "changed": changed,
            "number": number,
        })
        if approved_at is not UNSET:
            field_dict["approvedAt"] = approved_at
        if sent_at is not UNSET:
            field_dict["sentAt"] = sent_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        changed = cast(list[str], d.pop("changed"))


        number = d.pop("number")

        _approved_at = d.pop("approvedAt", UNSET)
        approved_at: datetime.datetime | Unset
        if isinstance(_approved_at,  Unset):
            approved_at = UNSET
        else:
            approved_at = datetime.datetime.fromisoformat(_approved_at)




        _sent_at = d.pop("sentAt", UNSET)
        sent_at: datetime.datetime | Unset
        if isinstance(_sent_at,  Unset):
            sent_at = UNSET
        else:
            sent_at = datetime.datetime.fromisoformat(_sent_at)




        digest_issue_since = cls(
            changed=changed,
            number=number,
            approved_at=approved_at,
            sent_at=sent_at,
        )

        return digest_issue_since

