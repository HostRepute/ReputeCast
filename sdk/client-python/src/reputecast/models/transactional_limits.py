from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.transactional_provider_limit import TransactionalProviderLimit





T = TypeVar("T", bound="TransactionalLimits")



@_attrs_define
class TransactionalLimits:
    """ 
        Attributes:
            idempotency_hours (int):
            max_batch (int):
            rate_per_second (int):
            route_limit_bytes (int):
            route_limit_others (list[TransactionalProviderLimit]):
            schedule_hours (int):
            smtp_recipients (int):
            route_limit_provider (str | Unset):
     """

    idempotency_hours: int
    max_batch: int
    rate_per_second: int
    route_limit_bytes: int
    route_limit_others: list[TransactionalProviderLimit]
    schedule_hours: int
    smtp_recipients: int
    route_limit_provider: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.transactional_provider_limit import TransactionalProviderLimit # noqa: PLC0415
        idempotency_hours = self.idempotency_hours

        max_batch = self.max_batch

        rate_per_second = self.rate_per_second

        route_limit_bytes = self.route_limit_bytes

        route_limit_others = []
        for route_limit_others_item_data in self.route_limit_others:
            route_limit_others_item = route_limit_others_item_data.to_dict()
            route_limit_others.append(route_limit_others_item)



        schedule_hours = self.schedule_hours

        smtp_recipients = self.smtp_recipients

        route_limit_provider = self.route_limit_provider


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "idempotencyHours": idempotency_hours,
            "maxBatch": max_batch,
            "ratePerSecond": rate_per_second,
            "routeLimitBytes": route_limit_bytes,
            "routeLimitOthers": route_limit_others,
            "scheduleHours": schedule_hours,
            "smtpRecipients": smtp_recipients,
        })
        if route_limit_provider is not UNSET:
            field_dict["routeLimitProvider"] = route_limit_provider

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.transactional_provider_limit import TransactionalProviderLimit # noqa: PLC0415
        d = dict(src_dict)
        idempotency_hours = d.pop("idempotencyHours")

        max_batch = d.pop("maxBatch")

        rate_per_second = d.pop("ratePerSecond")

        route_limit_bytes = d.pop("routeLimitBytes")

        route_limit_others = []
        _route_limit_others = d.pop("routeLimitOthers")
        for route_limit_others_item_data in (_route_limit_others):
            route_limit_others_item = TransactionalProviderLimit.from_dict(route_limit_others_item_data)



            route_limit_others.append(route_limit_others_item)


        schedule_hours = d.pop("scheduleHours")

        smtp_recipients = d.pop("smtpRecipients")

        route_limit_provider = d.pop("routeLimitProvider", UNSET)

        transactional_limits = cls(
            idempotency_hours=idempotency_hours,
            max_batch=max_batch,
            rate_per_second=rate_per_second,
            route_limit_bytes=route_limit_bytes,
            route_limit_others=route_limit_others,
            schedule_hours=schedule_hours,
            smtp_recipients=smtp_recipients,
            route_limit_provider=route_limit_provider,
        )

        return transactional_limits

