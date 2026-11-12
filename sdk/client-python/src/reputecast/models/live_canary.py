from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.live_canary_phase import LiveCanaryPhase
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.canary_gate import CanaryGate
  from ..models.canary_ip import CanaryIP
  from ..models.seed_placement import SeedPlacement





T = TypeVar("T", bound="LiveCanary")



@_attrs_define
class LiveCanary:
    """ 
        Attributes:
            chosen (bool):
            domain (str):
            extended (int):
            gates (list[CanaryGate]):
            main_seconds (int):
            never_mailed (int):
            new_ips (list[CanaryIP]):
            percent (float):
            phase (LiveCanaryPhase):
            queued (int):
            reasons (list[str]):
            seeds (int):
            sent (int):
            size (int):
            usual (int):
            watch_minutes (int):
            decided_at (datetime.datetime | Unset):
            decided_by (str | Unset):
            placement (SeedPlacement | Unset):
            sent_at (datetime.datetime | Unset):
            started_at (datetime.datetime | Unset):
            until (datetime.datetime | Unset):
     """

    chosen: bool
    domain: str
    extended: int
    gates: list[CanaryGate]
    main_seconds: int
    never_mailed: int
    new_ips: list[CanaryIP]
    percent: float
    phase: LiveCanaryPhase
    queued: int
    reasons: list[str]
    seeds: int
    sent: int
    size: int
    usual: int
    watch_minutes: int
    decided_at: datetime.datetime | Unset = UNSET
    decided_by: str | Unset = UNSET
    placement: SeedPlacement | Unset = UNSET
    sent_at: datetime.datetime | Unset = UNSET
    started_at: datetime.datetime | Unset = UNSET
    until: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.canary_gate import CanaryGate # noqa: PLC0415
        from ..models.canary_ip import CanaryIP # noqa: PLC0415
        from ..models.seed_placement import SeedPlacement # noqa: PLC0415
        chosen = self.chosen

        domain = self.domain

        extended = self.extended

        gates = []
        for gates_item_data in self.gates:
            gates_item = gates_item_data.to_dict()
            gates.append(gates_item)



        main_seconds = self.main_seconds

        never_mailed = self.never_mailed

        new_ips = []
        for new_ips_item_data in self.new_ips:
            new_ips_item = new_ips_item_data.to_dict()
            new_ips.append(new_ips_item)



        percent = self.percent

        phase = self.phase.value

        queued = self.queued

        reasons = self.reasons



        seeds = self.seeds

        sent = self.sent

        size = self.size

        usual = self.usual

        watch_minutes = self.watch_minutes

        decided_at: str | Unset = UNSET
        if not isinstance(self.decided_at, Unset):
            decided_at = self.decided_at.isoformat()

        decided_by = self.decided_by

        placement: dict[str, Any] | Unset = UNSET
        if not isinstance(self.placement, Unset):
            placement = self.placement.to_dict()

        sent_at: str | Unset = UNSET
        if not isinstance(self.sent_at, Unset):
            sent_at = self.sent_at.isoformat()

        started_at: str | Unset = UNSET
        if not isinstance(self.started_at, Unset):
            started_at = self.started_at.isoformat()

        until: str | Unset = UNSET
        if not isinstance(self.until, Unset):
            until = self.until.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "chosen": chosen,
            "domain": domain,
            "extended": extended,
            "gates": gates,
            "mainSeconds": main_seconds,
            "neverMailed": never_mailed,
            "newIps": new_ips,
            "percent": percent,
            "phase": phase,
            "queued": queued,
            "reasons": reasons,
            "seeds": seeds,
            "sent": sent,
            "size": size,
            "usual": usual,
            "watchMinutes": watch_minutes,
        })
        if decided_at is not UNSET:
            field_dict["decidedAt"] = decided_at
        if decided_by is not UNSET:
            field_dict["decidedBy"] = decided_by
        if placement is not UNSET:
            field_dict["placement"] = placement
        if sent_at is not UNSET:
            field_dict["sentAt"] = sent_at
        if started_at is not UNSET:
            field_dict["startedAt"] = started_at
        if until is not UNSET:
            field_dict["until"] = until

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.canary_gate import CanaryGate # noqa: PLC0415
        from ..models.canary_ip import CanaryIP # noqa: PLC0415
        from ..models.seed_placement import SeedPlacement # noqa: PLC0415
        d = dict(src_dict)
        chosen = d.pop("chosen")

        domain = d.pop("domain")

        extended = d.pop("extended")

        gates = []
        _gates = d.pop("gates")
        for gates_item_data in (_gates):
            gates_item = CanaryGate.from_dict(gates_item_data)



            gates.append(gates_item)


        main_seconds = d.pop("mainSeconds")

        never_mailed = d.pop("neverMailed")

        new_ips = []
        _new_ips = d.pop("newIps")
        for new_ips_item_data in (_new_ips):
            new_ips_item = CanaryIP.from_dict(new_ips_item_data)



            new_ips.append(new_ips_item)


        percent = d.pop("percent")

        phase = LiveCanaryPhase(d.pop("phase"))




        queued = d.pop("queued")

        reasons = cast(list[str], d.pop("reasons"))


        seeds = d.pop("seeds")

        sent = d.pop("sent")

        size = d.pop("size")

        usual = d.pop("usual")

        watch_minutes = d.pop("watchMinutes")

        _decided_at = d.pop("decidedAt", UNSET)
        decided_at: datetime.datetime | Unset
        if isinstance(_decided_at,  Unset):
            decided_at = UNSET
        else:
            decided_at = datetime.datetime.fromisoformat(_decided_at)




        decided_by = d.pop("decidedBy", UNSET)

        _placement = d.pop("placement", UNSET)
        placement: SeedPlacement | Unset
        if isinstance(_placement,  Unset):
            placement = UNSET
        else:
            placement = SeedPlacement.from_dict(_placement)




        _sent_at = d.pop("sentAt", UNSET)
        sent_at: datetime.datetime | Unset
        if isinstance(_sent_at,  Unset):
            sent_at = UNSET
        else:
            sent_at = datetime.datetime.fromisoformat(_sent_at)




        _started_at = d.pop("startedAt", UNSET)
        started_at: datetime.datetime | Unset
        if isinstance(_started_at,  Unset):
            started_at = UNSET
        else:
            started_at = datetime.datetime.fromisoformat(_started_at)




        _until = d.pop("until", UNSET)
        until: datetime.datetime | Unset
        if isinstance(_until,  Unset):
            until = UNSET
        else:
            until = datetime.datetime.fromisoformat(_until)




        live_canary = cls(
            chosen=chosen,
            domain=domain,
            extended=extended,
            gates=gates,
            main_seconds=main_seconds,
            never_mailed=never_mailed,
            new_ips=new_ips,
            percent=percent,
            phase=phase,
            queued=queued,
            reasons=reasons,
            seeds=seeds,
            sent=sent,
            size=size,
            usual=usual,
            watch_minutes=watch_minutes,
            decided_at=decided_at,
            decided_by=decided_by,
            placement=placement,
            sent_at=sent_at,
            started_at=started_at,
            until=until,
        )

        return live_canary

