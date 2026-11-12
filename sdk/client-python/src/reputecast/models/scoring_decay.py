from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ScoringDecay")



@_attrs_define
class ScoringDecay:
    """ 
        Attributes:
            after_weeks (int):
            enabled (bool):
            percent (int):
     """

    after_weeks: int
    enabled: bool
    percent: int





    def to_dict(self) -> dict[str, Any]:
        after_weeks = self.after_weeks

        enabled = self.enabled

        percent = self.percent


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "afterWeeks": after_weeks,
            "enabled": enabled,
            "percent": percent,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        after_weeks = d.pop("afterWeeks")

        enabled = d.pop("enabled")

        percent = d.pop("percent")

        scoring_decay = cls(
            after_weeks=after_weeks,
            enabled=enabled,
            percent=percent,
        )

        return scoring_decay

