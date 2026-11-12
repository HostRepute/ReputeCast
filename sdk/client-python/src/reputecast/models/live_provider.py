from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.live_provider_color import LiveProviderColor
from ..models.live_provider_route_status import LiveProviderRouteStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.live_warmup_hold import LiveWarmupHold





T = TypeVar("T", bound="LiveProvider")



@_attrs_define
class LiveProvider:
    """ 
        Attributes:
            adapter (str):
            color (LiveProviderColor):
            cost (float):
            host (str):
            id (str):
            idempotent (bool):
            infers_delivery (bool):
            ip (str):
            name (str):
            planned (int):
            planned_cost (float):
            probe_seed (bool):
            probes (int):
            probes_needed (int):
            probing (bool):
            rate (float):
            remaining (int):
            retrying (bool):
            route_status (LiveProviderRouteStatus):
            sent (int):
            share (float):
            short (str):
            status (str):
            api_error_rate (float | Unset):
            bounce_rate (float | Unset):
            complaint_rate (float | Unset):
            delivered_rate (float | Unset):
            done_at (datetime.datetime | Unset):
            down_since (datetime.datetime | Unset):
            idempotency_seconds (int | Unset):
            next_probe (datetime.datetime | Unset):
            quota_free (int | Unset):
            quota_max (int | Unset):
            retry_until (datetime.datetime | Unset):
            warmup_cap (int | Unset):
            warmup_day (int | Unset):
            warmup_days (int | Unset):
            warmup_hold (LiveWarmupHold | Unset):
            warmup_sent (int | Unset):
     """

    adapter: str
    color: LiveProviderColor
    cost: float
    host: str
    id: str
    idempotent: bool
    infers_delivery: bool
    ip: str
    name: str
    planned: int
    planned_cost: float
    probe_seed: bool
    probes: int
    probes_needed: int
    probing: bool
    rate: float
    remaining: int
    retrying: bool
    route_status: LiveProviderRouteStatus
    sent: int
    share: float
    short: str
    status: str
    api_error_rate: float | Unset = UNSET
    bounce_rate: float | Unset = UNSET
    complaint_rate: float | Unset = UNSET
    delivered_rate: float | Unset = UNSET
    done_at: datetime.datetime | Unset = UNSET
    down_since: datetime.datetime | Unset = UNSET
    idempotency_seconds: int | Unset = UNSET
    next_probe: datetime.datetime | Unset = UNSET
    quota_free: int | Unset = UNSET
    quota_max: int | Unset = UNSET
    retry_until: datetime.datetime | Unset = UNSET
    warmup_cap: int | Unset = UNSET
    warmup_day: int | Unset = UNSET
    warmup_days: int | Unset = UNSET
    warmup_hold: LiveWarmupHold | Unset = UNSET
    warmup_sent: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.live_warmup_hold import LiveWarmupHold # noqa: PLC0415
        adapter = self.adapter

        color = self.color.value

        cost = self.cost

        host = self.host

        id = self.id

        idempotent = self.idempotent

        infers_delivery = self.infers_delivery

        ip = self.ip

        name = self.name

        planned = self.planned

        planned_cost = self.planned_cost

        probe_seed = self.probe_seed

        probes = self.probes

        probes_needed = self.probes_needed

        probing = self.probing

        rate = self.rate

        remaining = self.remaining

        retrying = self.retrying

        route_status = self.route_status.value

        sent = self.sent

        share = self.share

        short = self.short

        status = self.status

        api_error_rate = self.api_error_rate

        bounce_rate = self.bounce_rate

        complaint_rate = self.complaint_rate

        delivered_rate = self.delivered_rate

        done_at: str | Unset = UNSET
        if not isinstance(self.done_at, Unset):
            done_at = self.done_at.isoformat()

        down_since: str | Unset = UNSET
        if not isinstance(self.down_since, Unset):
            down_since = self.down_since.isoformat()

        idempotency_seconds = self.idempotency_seconds

        next_probe: str | Unset = UNSET
        if not isinstance(self.next_probe, Unset):
            next_probe = self.next_probe.isoformat()

        quota_free = self.quota_free

        quota_max = self.quota_max

        retry_until: str | Unset = UNSET
        if not isinstance(self.retry_until, Unset):
            retry_until = self.retry_until.isoformat()

        warmup_cap = self.warmup_cap

        warmup_day = self.warmup_day

        warmup_days = self.warmup_days

        warmup_hold: dict[str, Any] | Unset = UNSET
        if not isinstance(self.warmup_hold, Unset):
            warmup_hold = self.warmup_hold.to_dict()

        warmup_sent = self.warmup_sent


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "adapter": adapter,
            "color": color,
            "cost": cost,
            "host": host,
            "id": id,
            "idempotent": idempotent,
            "infersDelivery": infers_delivery,
            "ip": ip,
            "name": name,
            "planned": planned,
            "plannedCost": planned_cost,
            "probeSeed": probe_seed,
            "probes": probes,
            "probesNeeded": probes_needed,
            "probing": probing,
            "rate": rate,
            "remaining": remaining,
            "retrying": retrying,
            "routeStatus": route_status,
            "sent": sent,
            "share": share,
            "short": short,
            "status": status,
        })
        if api_error_rate is not UNSET:
            field_dict["apiErrorRate"] = api_error_rate
        if bounce_rate is not UNSET:
            field_dict["bounceRate"] = bounce_rate
        if complaint_rate is not UNSET:
            field_dict["complaintRate"] = complaint_rate
        if delivered_rate is not UNSET:
            field_dict["deliveredRate"] = delivered_rate
        if done_at is not UNSET:
            field_dict["doneAt"] = done_at
        if down_since is not UNSET:
            field_dict["downSince"] = down_since
        if idempotency_seconds is not UNSET:
            field_dict["idempotencySeconds"] = idempotency_seconds
        if next_probe is not UNSET:
            field_dict["nextProbe"] = next_probe
        if quota_free is not UNSET:
            field_dict["quotaFree"] = quota_free
        if quota_max is not UNSET:
            field_dict["quotaMax"] = quota_max
        if retry_until is not UNSET:
            field_dict["retryUntil"] = retry_until
        if warmup_cap is not UNSET:
            field_dict["warmupCap"] = warmup_cap
        if warmup_day is not UNSET:
            field_dict["warmupDay"] = warmup_day
        if warmup_days is not UNSET:
            field_dict["warmupDays"] = warmup_days
        if warmup_hold is not UNSET:
            field_dict["warmupHold"] = warmup_hold
        if warmup_sent is not UNSET:
            field_dict["warmupSent"] = warmup_sent

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.live_warmup_hold import LiveWarmupHold # noqa: PLC0415
        d = dict(src_dict)
        adapter = d.pop("adapter")

        color = LiveProviderColor(d.pop("color"))




        cost = d.pop("cost")

        host = d.pop("host")

        id = d.pop("id")

        idempotent = d.pop("idempotent")

        infers_delivery = d.pop("infersDelivery")

        ip = d.pop("ip")

        name = d.pop("name")

        planned = d.pop("planned")

        planned_cost = d.pop("plannedCost")

        probe_seed = d.pop("probeSeed")

        probes = d.pop("probes")

        probes_needed = d.pop("probesNeeded")

        probing = d.pop("probing")

        rate = d.pop("rate")

        remaining = d.pop("remaining")

        retrying = d.pop("retrying")

        route_status = LiveProviderRouteStatus(d.pop("routeStatus"))




        sent = d.pop("sent")

        share = d.pop("share")

        short = d.pop("short")

        status = d.pop("status")

        api_error_rate = d.pop("apiErrorRate", UNSET)

        bounce_rate = d.pop("bounceRate", UNSET)

        complaint_rate = d.pop("complaintRate", UNSET)

        delivered_rate = d.pop("deliveredRate", UNSET)

        _done_at = d.pop("doneAt", UNSET)
        done_at: datetime.datetime | Unset
        if isinstance(_done_at,  Unset):
            done_at = UNSET
        else:
            done_at = datetime.datetime.fromisoformat(_done_at)




        _down_since = d.pop("downSince", UNSET)
        down_since: datetime.datetime | Unset
        if isinstance(_down_since,  Unset):
            down_since = UNSET
        else:
            down_since = datetime.datetime.fromisoformat(_down_since)




        idempotency_seconds = d.pop("idempotencySeconds", UNSET)

        _next_probe = d.pop("nextProbe", UNSET)
        next_probe: datetime.datetime | Unset
        if isinstance(_next_probe,  Unset):
            next_probe = UNSET
        else:
            next_probe = datetime.datetime.fromisoformat(_next_probe)




        quota_free = d.pop("quotaFree", UNSET)

        quota_max = d.pop("quotaMax", UNSET)

        _retry_until = d.pop("retryUntil", UNSET)
        retry_until: datetime.datetime | Unset
        if isinstance(_retry_until,  Unset):
            retry_until = UNSET
        else:
            retry_until = datetime.datetime.fromisoformat(_retry_until)




        warmup_cap = d.pop("warmupCap", UNSET)

        warmup_day = d.pop("warmupDay", UNSET)

        warmup_days = d.pop("warmupDays", UNSET)

        _warmup_hold = d.pop("warmupHold", UNSET)
        warmup_hold: LiveWarmupHold | Unset
        if isinstance(_warmup_hold,  Unset):
            warmup_hold = UNSET
        else:
            warmup_hold = LiveWarmupHold.from_dict(_warmup_hold)




        warmup_sent = d.pop("warmupSent", UNSET)

        live_provider = cls(
            adapter=adapter,
            color=color,
            cost=cost,
            host=host,
            id=id,
            idempotent=idempotent,
            infers_delivery=infers_delivery,
            ip=ip,
            name=name,
            planned=planned,
            planned_cost=planned_cost,
            probe_seed=probe_seed,
            probes=probes,
            probes_needed=probes_needed,
            probing=probing,
            rate=rate,
            remaining=remaining,
            retrying=retrying,
            route_status=route_status,
            sent=sent,
            share=share,
            short=short,
            status=status,
            api_error_rate=api_error_rate,
            bounce_rate=bounce_rate,
            complaint_rate=complaint_rate,
            delivered_rate=delivered_rate,
            done_at=done_at,
            down_since=down_since,
            idempotency_seconds=idempotency_seconds,
            next_probe=next_probe,
            quota_free=quota_free,
            quota_max=quota_max,
            retry_until=retry_until,
            warmup_cap=warmup_cap,
            warmup_day=warmup_day,
            warmup_days=warmup_days,
            warmup_hold=warmup_hold,
            warmup_sent=warmup_sent,
        )

        return live_provider

