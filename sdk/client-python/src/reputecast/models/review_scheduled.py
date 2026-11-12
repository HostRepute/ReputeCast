from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="ReviewScheduled")



@_attrs_define
class ReviewScheduled:
    """ 
        Attributes:
            at (datetime.datetime):
            by (str):
     """

    at: datetime.datetime
    by: str





    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        by = self.by


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "by": by,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        by = d.pop("by")

        review_scheduled = cls(
            at=at,
            by=by,
        )

        return review_scheduled

