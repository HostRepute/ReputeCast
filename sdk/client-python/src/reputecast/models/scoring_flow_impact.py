from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.scoring_flow_impact_direction import ScoringFlowImpactDirection






T = TypeVar("T", bound="ScoringFlowImpact")



@_attrs_define
class ScoringFlowImpact:
    """ 
        Attributes:
            crossed (int):
            direction (ScoringFlowImpactDirection):
            id (str):
            name (str):
            started (int):
            threshold (int):
     """

    crossed: int
    direction: ScoringFlowImpactDirection
    id: str
    name: str
    started: int
    threshold: int





    def to_dict(self) -> dict[str, Any]:
        crossed = self.crossed

        direction = self.direction.value

        id = self.id

        name = self.name

        started = self.started

        threshold = self.threshold


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "crossed": crossed,
            "direction": direction,
            "id": id,
            "name": name,
            "started": started,
            "threshold": threshold,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        crossed = d.pop("crossed")

        direction = ScoringFlowImpactDirection(d.pop("direction"))




        id = d.pop("id")

        name = d.pop("name")

        started = d.pop("started")

        threshold = d.pop("threshold")

        scoring_flow_impact = cls(
            crossed=crossed,
            direction=direction,
            id=id,
            name=name,
            started=started,
            threshold=threshold,
        )

        return scoring_flow_impact

