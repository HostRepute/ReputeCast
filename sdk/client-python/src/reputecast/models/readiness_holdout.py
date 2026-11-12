from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ReadinessHoldout")



@_attrs_define
class ReadinessHoldout:
    """ 
        Attributes:
            included (bool):
            left_out (int):
            percent (float):
            reached (int):
     """

    included: bool
    left_out: int
    percent: float
    reached: int





    def to_dict(self) -> dict[str, Any]:
        included = self.included

        left_out = self.left_out

        percent = self.percent

        reached = self.reached


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "included": included,
            "leftOut": left_out,
            "percent": percent,
            "reached": reached,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        included = d.pop("included")

        left_out = d.pop("leftOut")

        percent = d.pop("percent")

        reached = d.pop("reached")

        readiness_holdout = cls(
            included=included,
            left_out=left_out,
            percent=percent,
            reached=reached,
        )

        return readiness_holdout

