from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.readiness_check import ReadinessCheck





T = TypeVar("T", bound="ComposerReadinessGroup")



@_attrs_define
class ComposerReadinessGroup:
    """ 
        Attributes:
            blocks (int):
            checks (list[ReadinessCheck]):
            warnings (int):
     """

    blocks: int
    checks: list[ReadinessCheck]
    warnings: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.readiness_check import ReadinessCheck # noqa: PLC0415
        blocks = self.blocks

        checks = []
        for checks_item_data in self.checks:
            checks_item = checks_item_data.to_dict()
            checks.append(checks_item)



        warnings = self.warnings


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "blocks": blocks,
            "checks": checks,
            "warnings": warnings,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.readiness_check import ReadinessCheck # noqa: PLC0415
        d = dict(src_dict)
        blocks = d.pop("blocks")

        checks = []
        _checks = d.pop("checks")
        for checks_item_data in (_checks):
            checks_item = ReadinessCheck.from_dict(checks_item_data)



            checks.append(checks_item)


        warnings = d.pop("warnings")

        composer_readiness_group = cls(
            blocks=blocks,
            checks=checks,
            warnings=warnings,
        )

        return composer_readiness_group

