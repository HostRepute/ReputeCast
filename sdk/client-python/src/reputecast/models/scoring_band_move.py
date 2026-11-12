from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.scoring_band_move_from import ScoringBandMoveFrom
from ..models.scoring_band_move_to import ScoringBandMoveTo






T = TypeVar("T", bound="ScoringBandMove")



@_attrs_define
class ScoringBandMove:
    """ 
        Attributes:
            count (int):
            from_ (ScoringBandMoveFrom):
            to (ScoringBandMoveTo):
     """

    count: int
    from_: ScoringBandMoveFrom
    to: ScoringBandMoveTo





    def to_dict(self) -> dict[str, Any]:
        count = self.count

        from_ = self.from_.value

        to = self.to.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "count": count,
            "from": from_,
            "to": to,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        count = d.pop("count")

        from_ = ScoringBandMoveFrom(d.pop("from"))




        to = ScoringBandMoveTo(d.pop("to"))




        scoring_band_move = cls(
            count=count,
            from_=from_,
            to=to,
        )

        return scoring_band_move

