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
  from ..models.scoring_bucket import ScoringBucket
  from ..models.scoring_flow_impact import ScoringFlowImpact
  from ..models.scoring_impact_field_matches import ScoringImpactFieldMatches





T = TypeVar("T", bound="ScoringImpact")



@_attrs_define
class ScoringImpact:
    """ 
        Attributes:
            automations (list[ScoringFlowImpact]):
            bands_after (ScoringBandCounts):
            bands_before (ScoringBandCounts):
            changed (int):
            contacts (int):
            current (list[ScoringBucket]):
            duration_ms (int):
            estimated (bool):
            field_matches (ScoringImpactFieldMatches):
            moves (list[ScoringBandMove]):
            rescored (int):
            spread (list[ScoringBucket]):
     """

    automations: list[ScoringFlowImpact]
    bands_after: ScoringBandCounts
    bands_before: ScoringBandCounts
    changed: int
    contacts: int
    current: list[ScoringBucket]
    duration_ms: int
    estimated: bool
    field_matches: ScoringImpactFieldMatches
    moves: list[ScoringBandMove]
    rescored: int
    spread: list[ScoringBucket]





    def to_dict(self) -> dict[str, Any]:
        from ..models.scoring_band_counts import ScoringBandCounts # noqa: PLC0415
        from ..models.scoring_band_move import ScoringBandMove # noqa: PLC0415
        from ..models.scoring_bucket import ScoringBucket # noqa: PLC0415
        from ..models.scoring_flow_impact import ScoringFlowImpact # noqa: PLC0415
        from ..models.scoring_impact_field_matches import ScoringImpactFieldMatches # noqa: PLC0415
        automations = []
        for automations_item_data in self.automations:
            automations_item = automations_item_data.to_dict()
            automations.append(automations_item)



        bands_after = self.bands_after.to_dict()

        bands_before = self.bands_before.to_dict()

        changed = self.changed

        contacts = self.contacts

        current = []
        for current_item_data in self.current:
            current_item = current_item_data.to_dict()
            current.append(current_item)



        duration_ms = self.duration_ms

        estimated = self.estimated

        field_matches = self.field_matches.to_dict()

        moves = []
        for moves_item_data in self.moves:
            moves_item = moves_item_data.to_dict()
            moves.append(moves_item)



        rescored = self.rescored

        spread = []
        for spread_item_data in self.spread:
            spread_item = spread_item_data.to_dict()
            spread.append(spread_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "automations": automations,
            "bandsAfter": bands_after,
            "bandsBefore": bands_before,
            "changed": changed,
            "contacts": contacts,
            "current": current,
            "durationMs": duration_ms,
            "estimated": estimated,
            "fieldMatches": field_matches,
            "moves": moves,
            "rescored": rescored,
            "spread": spread,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.scoring_band_counts import ScoringBandCounts # noqa: PLC0415
        from ..models.scoring_band_move import ScoringBandMove # noqa: PLC0415
        from ..models.scoring_bucket import ScoringBucket # noqa: PLC0415
        from ..models.scoring_flow_impact import ScoringFlowImpact # noqa: PLC0415
        from ..models.scoring_impact_field_matches import ScoringImpactFieldMatches # noqa: PLC0415
        d = dict(src_dict)
        automations = []
        _automations = d.pop("automations")
        for automations_item_data in (_automations):
            automations_item = ScoringFlowImpact.from_dict(automations_item_data)



            automations.append(automations_item)


        bands_after = ScoringBandCounts.from_dict(d.pop("bandsAfter"))




        bands_before = ScoringBandCounts.from_dict(d.pop("bandsBefore"))




        changed = d.pop("changed")

        contacts = d.pop("contacts")

        current = []
        _current = d.pop("current")
        for current_item_data in (_current):
            current_item = ScoringBucket.from_dict(current_item_data)



            current.append(current_item)


        duration_ms = d.pop("durationMs")

        estimated = d.pop("estimated")

        field_matches = ScoringImpactFieldMatches.from_dict(d.pop("fieldMatches"))




        moves = []
        _moves = d.pop("moves")
        for moves_item_data in (_moves):
            moves_item = ScoringBandMove.from_dict(moves_item_data)



            moves.append(moves_item)


        rescored = d.pop("rescored")

        spread = []
        _spread = d.pop("spread")
        for spread_item_data in (_spread):
            spread_item = ScoringBucket.from_dict(spread_item_data)



            spread.append(spread_item)


        scoring_impact = cls(
            automations=automations,
            bands_after=bands_after,
            bands_before=bands_before,
            changed=changed,
            contacts=contacts,
            current=current,
            duration_ms=duration_ms,
            estimated=estimated,
            field_matches=field_matches,
            moves=moves,
            rescored=rescored,
            spread=spread,
        )

        return scoring_impact

