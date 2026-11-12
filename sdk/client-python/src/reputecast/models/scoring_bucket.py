from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ScoringBucket")



@_attrs_define
class ScoringBucket:
    """ 
        Attributes:
            contacts (int):
            from_ (int):
            to (int):
     """

    contacts: int
    from_: int
    to: int





    def to_dict(self) -> dict[str, Any]:
        contacts = self.contacts

        from_ = self.from_

        to = self.to


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "contacts": contacts,
            "from": from_,
            "to": to,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        contacts = d.pop("contacts")

        from_ = d.pop("from")

        to = d.pop("to")

        scoring_bucket = cls(
            contacts=contacts,
            from_=from_,
            to=to,
        )

        return scoring_bucket

