from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="WhatsAppSuppression")



@_attrs_define
class WhatsAppSuppression:
    """ 
        Attributes:
            at (datetime.datetime):
            reason (str):
            keyword (str | Unset):
     """

    at: datetime.datetime
    reason: str
    keyword: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        reason = self.reason

        keyword = self.keyword


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "reason": reason,
        })
        if keyword is not UNSET:
            field_dict["keyword"] = keyword

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        reason = d.pop("reason")

        keyword = d.pop("keyword", UNSET)

        whats_app_suppression = cls(
            at=at,
            reason=reason,
            keyword=keyword,
        )

        return whats_app_suppression

