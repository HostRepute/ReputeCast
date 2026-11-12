from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.vet_check import VetCheck





T = TypeVar("T", bound="WarmupHoldAlternative")



@_attrs_define
class WarmupHoldAlternative:
    """ 
        Attributes:
            allowed (bool):
            checks (list[VetCheck]):
            name (str):
            provider_id (str):
            short (str):
     """

    allowed: bool
    checks: list[VetCheck]
    name: str
    provider_id: str
    short: str





    def to_dict(self) -> dict[str, Any]:
        from ..models.vet_check import VetCheck # noqa: PLC0415
        allowed = self.allowed

        checks = []
        for checks_item_data in self.checks:
            checks_item = checks_item_data.to_dict()
            checks.append(checks_item)



        name = self.name

        provider_id = self.provider_id

        short = self.short


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "allowed": allowed,
            "checks": checks,
            "name": name,
            "providerId": provider_id,
            "short": short,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.vet_check import VetCheck # noqa: PLC0415
        d = dict(src_dict)
        allowed = d.pop("allowed")

        checks = []
        _checks = d.pop("checks")
        for checks_item_data in (_checks):
            checks_item = VetCheck.from_dict(checks_item_data)



            checks.append(checks_item)


        name = d.pop("name")

        provider_id = d.pop("providerId")

        short = d.pop("short")

        warmup_hold_alternative = cls(
            allowed=allowed,
            checks=checks,
            name=name,
            provider_id=provider_id,
            short=short,
        )

        return warmup_hold_alternative

