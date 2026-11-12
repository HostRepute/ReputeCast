from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.scoring_band_counts import ScoringBandCounts
  from ..models.scoring_band_move import ScoringBandMove
  from ..models.scoring_flow_impact import ScoringFlowImpact





T = TypeVar("T", bound="ScoringRecalcSummary")



@_attrs_define
class ScoringRecalcSummary:
    """ 
        Attributes:
            automations (list[ScoringFlowImpact]):
            bands (ScoringBandCounts):
            changed (int):
            moves (list[ScoringBandMove]):
     """

    automations: list[ScoringFlowImpact]
    bands: ScoringBandCounts
    changed: int
    moves: list[ScoringBandMove]





    def to_dict(self) -> dict[str, Any]:
        from ..models.scoring_band_counts import ScoringBandCounts # noqa: PLC0415
        from ..models.scoring_band_move import ScoringBandMove # noqa: PLC0415
        from ..models.scoring_flow_impact import ScoringFlowImpact # noqa: PLC0415
        automations = []
        for automations_item_data in self.automations:
            automations_item = automations_item_data.to_dict()
            automations.append(automations_item)



        bands = self.bands.to_dict()

        changed = self.changed

        moves = []
        for moves_item_data in self.moves:
            moves_item = moves_item_data.to_dict()
            moves.append(moves_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "automations": automations,
            "bands": bands,
            "changed": changed,
            "moves": moves,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.scoring_band_counts import ScoringBandCounts # noqa: PLC0415
        from ..models.scoring_band_move import ScoringBandMove # noqa: PLC0415
        from ..models.scoring_flow_impact import ScoringFlowImpact # noqa: PLC0415
        d = dict(src_dict)
        automations = []
        _automations = d.pop("automations")
        for automations_item_data in (_automations):
            automations_item = ScoringFlowImpact.from_dict(automations_item_data)



            automations.append(automations_item)


        bands = ScoringBandCounts.from_dict(d.pop("bands"))




        changed = d.pop("changed")

        moves = []
        _moves = d.pop("moves")
        for moves_item_data in (_moves):
            moves_item = ScoringBandMove.from_dict(moves_item_data)



            moves.append(moves_item)


        scoring_recalc_summary = cls(
            automations=automations,
            bands=bands,
            changed=changed,
            moves=moves,
        )

        return scoring_recalc_summary

