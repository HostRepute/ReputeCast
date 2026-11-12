from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.scoring_bands import ScoringBands
  from ..models.scoring_decay import ScoringDecay
  from ..models.scoring_rule import ScoringRule





T = TypeVar("T", bound="ScoringConfig")



@_attrs_define
class ScoringConfig:
    """ 
        Attributes:
            bands (ScoringBands):
            decay (ScoringDecay):
            rules (list[ScoringRule]):
     """

    bands: ScoringBands
    decay: ScoringDecay
    rules: list[ScoringRule]





    def to_dict(self) -> dict[str, Any]:
        from ..models.scoring_bands import ScoringBands # noqa: PLC0415
        from ..models.scoring_decay import ScoringDecay # noqa: PLC0415
        from ..models.scoring_rule import ScoringRule # noqa: PLC0415
        bands = self.bands.to_dict()

        decay = self.decay.to_dict()

        rules = []
        for rules_item_data in self.rules:
            rules_item = rules_item_data.to_dict()
            rules.append(rules_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "bands": bands,
            "decay": decay,
            "rules": rules,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.scoring_bands import ScoringBands # noqa: PLC0415
        from ..models.scoring_decay import ScoringDecay # noqa: PLC0415
        from ..models.scoring_rule import ScoringRule # noqa: PLC0415
        d = dict(src_dict)
        bands = ScoringBands.from_dict(d.pop("bands"))




        decay = ScoringDecay.from_dict(d.pop("decay"))




        rules = []
        _rules = d.pop("rules")
        for rules_item_data in (_rules):
            rules_item = ScoringRule.from_dict(rules_item_data)



            rules.append(rules_item)


        scoring_config = cls(
            bands=bands,
            decay=decay,
            rules=rules,
        )

        return scoring_config

