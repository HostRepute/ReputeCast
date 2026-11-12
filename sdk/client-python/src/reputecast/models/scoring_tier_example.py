from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ScoringTierExample")



@_attrs_define
class ScoringTierExample:
    """ 
        Attributes:
            id (str):
            name (str):
            score (int):
     """

    id: str
    name: str
    score: int





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        name = self.name

        score = self.score


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "name": name,
            "score": score,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        name = d.pop("name")

        score = d.pop("score")

        scoring_tier_example = cls(
            id=id,
            name=name,
            score=score,
        )

        return scoring_tier_example

