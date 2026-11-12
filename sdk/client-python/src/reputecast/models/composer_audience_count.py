from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ComposerAudienceCount")



@_attrs_define
class ComposerAudienceCount:
    """ 
        Attributes:
            approximate (bool):
            chose_rhythm (int):
            count (int):
     """

    approximate: bool
    chose_rhythm: int
    count: int





    def to_dict(self) -> dict[str, Any]:
        approximate = self.approximate

        chose_rhythm = self.chose_rhythm

        count = self.count


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "approximate": approximate,
            "choseRhythm": chose_rhythm,
            "count": count,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        approximate = d.pop("approximate")

        chose_rhythm = d.pop("choseRhythm")

        count = d.pop("count")

        composer_audience_count = cls(
            approximate=approximate,
            chose_rhythm=chose_rhythm,
            count=count,
        )

        return composer_audience_count

