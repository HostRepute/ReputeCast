from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="PollNPS")



@_attrs_define
class PollNPS:
    """ 
        Attributes:
            detractor_percent (float):
            detractors (int):
            passive_percent (float):
            passives (int):
            promoter_percent (float):
            promoters (int):
            score (int):
     """

    detractor_percent: float
    detractors: int
    passive_percent: float
    passives: int
    promoter_percent: float
    promoters: int
    score: int





    def to_dict(self) -> dict[str, Any]:
        detractor_percent = self.detractor_percent

        detractors = self.detractors

        passive_percent = self.passive_percent

        passives = self.passives

        promoter_percent = self.promoter_percent

        promoters = self.promoters

        score = self.score


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "detractorPercent": detractor_percent,
            "detractors": detractors,
            "passivePercent": passive_percent,
            "passives": passives,
            "promoterPercent": promoter_percent,
            "promoters": promoters,
            "score": score,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        detractor_percent = d.pop("detractorPercent")

        detractors = d.pop("detractors")

        passive_percent = d.pop("passivePercent")

        passives = d.pop("passives")

        promoter_percent = d.pop("promoterPercent")

        promoters = d.pop("promoters")

        score = d.pop("score")

        poll_nps = cls(
            detractor_percent=detractor_percent,
            detractors=detractors,
            passive_percent=passive_percent,
            passives=passives,
            promoter_percent=promoter_percent,
            promoters=promoters,
            score=score,
        )

        return poll_nps

