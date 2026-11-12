from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="ReviewCount")



@_attrs_define
class ReviewCount:
    """ 
        Attributes:
            at (datetime.datetime):
            recipients (int):
     """

    at: datetime.datetime
    recipients: int





    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        recipients = self.recipients


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "recipients": recipients,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        recipients = d.pop("recipients")

        review_count = cls(
            at=at,
            recipients=recipients,
        )

        return review_count

