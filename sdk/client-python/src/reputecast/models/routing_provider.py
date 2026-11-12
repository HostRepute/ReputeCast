from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.routing_provider_bulk import RoutingProviderBulk
from ..models.routing_provider_color import RoutingProviderColor
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.routing_share_step import RoutingShareStep
  from ..models.routing_terms import RoutingTerms





T = TypeVar("T", bound="RoutingProvider")



@_attrs_define
class RoutingProvider:
    """ 
        Attributes:
            adapter (str):
            broadcast_stream_creatable (bool):
            bulk (RoutingProviderBulk):
            bulk_allowed (bool):
            color (RoutingProviderColor):
            id (str):
            label (str):
            name (str):
            removes_webhook (bool):
            share_steps (list[RoutingShareStep]):
            short (str):
            status (str):
            terms (list[RoutingTerms]):
            warming_up (bool):
            warmup_day (int | Unset):
            warmup_days (int | Unset):
     """

    adapter: str
    broadcast_stream_creatable: bool
    bulk: RoutingProviderBulk
    bulk_allowed: bool
    color: RoutingProviderColor
    id: str
    label: str
    name: str
    removes_webhook: bool
    share_steps: list[RoutingShareStep]
    short: str
    status: str
    terms: list[RoutingTerms]
    warming_up: bool
    warmup_day: int | Unset = UNSET
    warmup_days: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.routing_share_step import RoutingShareStep # noqa: PLC0415
        from ..models.routing_terms import RoutingTerms # noqa: PLC0415
        adapter = self.adapter

        broadcast_stream_creatable = self.broadcast_stream_creatable

        bulk = self.bulk.value

        bulk_allowed = self.bulk_allowed

        color = self.color.value

        id = self.id

        label = self.label

        name = self.name

        removes_webhook = self.removes_webhook

        share_steps = []
        for share_steps_item_data in self.share_steps:
            share_steps_item = share_steps_item_data.to_dict()
            share_steps.append(share_steps_item)



        short = self.short

        status = self.status

        terms = []
        for terms_item_data in self.terms:
            terms_item = terms_item_data.to_dict()
            terms.append(terms_item)



        warming_up = self.warming_up

        warmup_day = self.warmup_day

        warmup_days = self.warmup_days


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "adapter": adapter,
            "broadcastStreamCreatable": broadcast_stream_creatable,
            "bulk": bulk,
            "bulkAllowed": bulk_allowed,
            "color": color,
            "id": id,
            "label": label,
            "name": name,
            "removesWebhook": removes_webhook,
            "shareSteps": share_steps,
            "short": short,
            "status": status,
            "terms": terms,
            "warmingUp": warming_up,
        })
        if warmup_day is not UNSET:
            field_dict["warmupDay"] = warmup_day
        if warmup_days is not UNSET:
            field_dict["warmupDays"] = warmup_days

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.routing_share_step import RoutingShareStep # noqa: PLC0415
        from ..models.routing_terms import RoutingTerms # noqa: PLC0415
        d = dict(src_dict)
        adapter = d.pop("adapter")

        broadcast_stream_creatable = d.pop("broadcastStreamCreatable")

        bulk = RoutingProviderBulk(d.pop("bulk"))




        bulk_allowed = d.pop("bulkAllowed")

        color = RoutingProviderColor(d.pop("color"))




        id = d.pop("id")

        label = d.pop("label")

        name = d.pop("name")

        removes_webhook = d.pop("removesWebhook")

        share_steps = []
        _share_steps = d.pop("shareSteps")
        for share_steps_item_data in (_share_steps):
            share_steps_item = RoutingShareStep.from_dict(share_steps_item_data)



            share_steps.append(share_steps_item)


        short = d.pop("short")

        status = d.pop("status")

        terms = []
        _terms = d.pop("terms")
        for terms_item_data in (_terms):
            terms_item = RoutingTerms.from_dict(terms_item_data)



            terms.append(terms_item)


        warming_up = d.pop("warmingUp")

        warmup_day = d.pop("warmupDay", UNSET)

        warmup_days = d.pop("warmupDays", UNSET)

        routing_provider = cls(
            adapter=adapter,
            broadcast_stream_creatable=broadcast_stream_creatable,
            bulk=bulk,
            bulk_allowed=bulk_allowed,
            color=color,
            id=id,
            label=label,
            name=name,
            removes_webhook=removes_webhook,
            share_steps=share_steps,
            short=short,
            status=status,
            terms=terms,
            warming_up=warming_up,
            warmup_day=warmup_day,
            warmup_days=warmup_days,
        )

        return routing_provider

