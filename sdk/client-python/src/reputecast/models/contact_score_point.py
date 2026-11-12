from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="ContactScorePoint")



@_attrs_define
class ContactScorePoint:
    """ 
        Attributes:
            day (str): YYYY-MM-DD in the workspace timezone
            score (int | Unset):
     """

    day: str
    score: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        day = self.day

        score = self.score


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "day": day,
        })
        if score is not UNSET:
            field_dict["score"] = score

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        day = d.pop("day")

        score = d.pop("score", UNSET)

        contact_score_point = cls(
            day=day,
            score=score,
        )

        return contact_score_point

