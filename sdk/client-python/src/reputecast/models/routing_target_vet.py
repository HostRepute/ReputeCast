from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.routing_vet_check import RoutingVetCheck





T = TypeVar("T", bound="RoutingTargetVet")



@_attrs_define
class RoutingTargetVet:
    """ 
        Attributes:
            checks (list[RoutingVetCheck]):
            provider (str):
            provider_id (str):
            warmup_day (int | Unset):
            warmup_days (int | Unset):
     """

    checks: list[RoutingVetCheck]
    provider: str
    provider_id: str
    warmup_day: int | Unset = UNSET
    warmup_days: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.routing_vet_check import RoutingVetCheck # noqa: PLC0415
        checks = []
        for checks_item_data in self.checks:
            checks_item = checks_item_data.to_dict()
            checks.append(checks_item)



        provider = self.provider

        provider_id = self.provider_id

        warmup_day = self.warmup_day

        warmup_days = self.warmup_days


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "checks": checks,
            "provider": provider,
            "providerId": provider_id,
        })
        if warmup_day is not UNSET:
            field_dict["warmupDay"] = warmup_day
        if warmup_days is not UNSET:
            field_dict["warmupDays"] = warmup_days

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.routing_vet_check import RoutingVetCheck # noqa: PLC0415
        d = dict(src_dict)
        checks = []
        _checks = d.pop("checks")
        for checks_item_data in (_checks):
            checks_item = RoutingVetCheck.from_dict(checks_item_data)



            checks.append(checks_item)


        provider = d.pop("provider")

        provider_id = d.pop("providerId")

        warmup_day = d.pop("warmupDay", UNSET)

        warmup_days = d.pop("warmupDays", UNSET)

        routing_target_vet = cls(
            checks=checks,
            provider=provider,
            provider_id=provider_id,
            warmup_day=warmup_day,
            warmup_days=warmup_days,
        )

        return routing_target_vet

