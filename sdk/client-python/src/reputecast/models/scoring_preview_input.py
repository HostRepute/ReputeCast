from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.scoring_config import ScoringConfig





T = TypeVar("T", bound="ScoringPreviewInput")



@_attrs_define
class ScoringPreviewInput:
    """ 
        Attributes:
            config (ScoringConfig):
     """

    config: ScoringConfig





    def to_dict(self) -> dict[str, Any]:
        from ..models.scoring_config import ScoringConfig # noqa: PLC0415
        config = self.config.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "config": config,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.scoring_config import ScoringConfig # noqa: PLC0415
        d = dict(src_dict)
        config = ScoringConfig.from_dict(d.pop("config"))




        scoring_preview_input = cls(
            config=config,
        )

        return scoring_preview_input

