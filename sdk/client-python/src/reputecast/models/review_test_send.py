from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ReviewTestSend")



@_attrs_define
class ReviewTestSend:
    """ 
        Attributes:
            at (datetime.datetime):
            by (str):
            provider (str):
            to (str):
            delivered (str | Unset):
            languages (list[str] | Unset):
     """

    at: datetime.datetime
    by: str
    provider: str
    to: str
    delivered: str | Unset = UNSET
    languages: list[str] | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        by = self.by

        provider = self.provider

        to = self.to

        delivered = self.delivered

        languages: list[str] | Unset = UNSET
        if not isinstance(self.languages, Unset):
            languages = self.languages




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "by": by,
            "provider": provider,
            "to": to,
        })
        if delivered is not UNSET:
            field_dict["delivered"] = delivered
        if languages is not UNSET:
            field_dict["languages"] = languages

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        by = d.pop("by")

        provider = d.pop("provider")

        to = d.pop("to")

        delivered = d.pop("delivered", UNSET)

        languages = cast(list[str], d.pop("languages", UNSET))


        review_test_send = cls(
            at=at,
            by=by,
            provider=provider,
            to=to,
            delivered=delivered,
            languages=languages,
        )

        return review_test_send

