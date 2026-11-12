from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.scoring_config import ScoringConfig





T = TypeVar("T", bound="ScoringSaveInput")



@_attrs_define
class ScoringSaveInput:
    """ 
        Attributes:
            base_version (int):
            config (ScoringConfig):
            start_automations (bool | Unset):
     """

    base_version: int
    config: ScoringConfig
    start_automations: bool | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.scoring_config import ScoringConfig # noqa: PLC0415
        base_version = self.base_version

        config = self.config.to_dict()

        start_automations = self.start_automations


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "baseVersion": base_version,
            "config": config,
        })
        if start_automations is not UNSET:
            field_dict["startAutomations"] = start_automations

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.scoring_config import ScoringConfig # noqa: PLC0415
        d = dict(src_dict)
        base_version = d.pop("baseVersion")

        config = ScoringConfig.from_dict(d.pop("config"))




        start_automations = d.pop("startAutomations", UNSET)

        scoring_save_input = cls(
            base_version=base_version,
            config=config,
            start_automations=start_automations,
        )

        return scoring_save_input

