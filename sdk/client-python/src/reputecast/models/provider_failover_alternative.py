from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.vet_check import VetCheck





T = TypeVar("T", bound="ProviderFailoverAlternative")



@_attrs_define
class ProviderFailoverAlternative:
    """ 
        Attributes:
            allowed (bool):
            checks (list[VetCheck]):
            exceeds_cap (bool):
            name (str):
            provider_id (str):
            recommended (bool):
            short (str):
            warming_up (bool):
     """

    allowed: bool
    checks: list[VetCheck]
    exceeds_cap: bool
    name: str
    provider_id: str
    recommended: bool
    short: str
    warming_up: bool





    def to_dict(self) -> dict[str, Any]:
        from ..models.vet_check import VetCheck # noqa: PLC0415
        allowed = self.allowed

        checks = []
        for checks_item_data in self.checks:
            checks_item = checks_item_data.to_dict()
            checks.append(checks_item)



        exceeds_cap = self.exceeds_cap

        name = self.name

        provider_id = self.provider_id

        recommended = self.recommended

        short = self.short

        warming_up = self.warming_up


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "allowed": allowed,
            "checks": checks,
            "exceedsCap": exceeds_cap,
            "name": name,
            "providerId": provider_id,
            "recommended": recommended,
            "short": short,
            "warmingUp": warming_up,
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


        exceeds_cap = d.pop("exceedsCap")

        name = d.pop("name")

        provider_id = d.pop("providerId")

        recommended = d.pop("recommended")

        short = d.pop("short")

        warming_up = d.pop("warmingUp")

        provider_failover_alternative = cls(
            allowed=allowed,
            checks=checks,
            exceeds_cap=exceeds_cap,
            name=name,
            provider_id=provider_id,
            recommended=recommended,
            short=short,
            warming_up=warming_up,
        )

        return provider_failover_alternative

