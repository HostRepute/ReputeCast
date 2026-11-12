from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ReviewHoldout")



@_attrs_define
class ReviewHoldout:
    """ 
        Attributes:
            included (bool):
            percent (float):
     """

    included: bool
    percent: float





    def to_dict(self) -> dict[str, Any]:
        included = self.included

        percent = self.percent


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "included": included,
            "percent": percent,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        included = d.pop("included")

        percent = d.pop("percent")

        review_holdout = cls(
            included=included,
            percent=percent,
        )

        return review_holdout

