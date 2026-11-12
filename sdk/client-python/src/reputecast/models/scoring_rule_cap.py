from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.scoring_rule_cap_kind import ScoringRuleCapKind
from ..types import UNSET, Unset






T = TypeVar("T", bound="ScoringRuleCap")



@_attrs_define
class ScoringRuleCap:
    """ 
        Attributes:
            kind (ScoringRuleCapKind):
            points (int | Unset):
     """

    kind: ScoringRuleCapKind
    points: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        kind = self.kind.value

        points = self.points


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "kind": kind,
        })
        if points is not UNSET:
            field_dict["points"] = points

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        kind = ScoringRuleCapKind(d.pop("kind"))




        points = d.pop("points", UNSET)

        scoring_rule_cap = cls(
            kind=kind,
            points=points,
        )

        return scoring_rule_cap

