from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ScoringBandCounts")



@_attrs_define
class ScoringBandCounts:
    """ 
        Attributes:
            cold (int):
            hot (int):
            warm (int):
     """

    cold: int
    hot: int
    warm: int





    def to_dict(self) -> dict[str, Any]:
        cold = self.cold

        hot = self.hot

        warm = self.warm


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "cold": cold,
            "hot": hot,
            "warm": warm,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        cold = d.pop("cold")

        hot = d.pop("hot")

        warm = d.pop("warm")

        scoring_band_counts = cls(
            cold=cold,
            hot=hot,
            warm=warm,
        )

        return scoring_band_counts

