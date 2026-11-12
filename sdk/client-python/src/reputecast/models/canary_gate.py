from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.canary_gate_key import CanaryGateKey
from ..models.canary_gate_state import CanaryGateState
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.seed_placement import SeedPlacement





T = TypeVar("T", bound="CanaryGate")



@_attrs_define
class CanaryGate:
    """ 
        Attributes:
            base (int):
            events (int):
            key (CanaryGateKey):
            limit (float):
            min_base (int):
            state (CanaryGateState):
            mailbox (str | Unset):
            not_reported (list[str] | Unset):
            reported (list[str] | Unset):
            seeds (SeedPlacement | Unset):
            value (float | Unset):
     """

    base: int
    events: int
    key: CanaryGateKey
    limit: float
    min_base: int
    state: CanaryGateState
    mailbox: str | Unset = UNSET
    not_reported: list[str] | Unset = UNSET
    reported: list[str] | Unset = UNSET
    seeds: SeedPlacement | Unset = UNSET
    value: float | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.seed_placement import SeedPlacement # noqa: PLC0415
        base = self.base

        events = self.events

        key = self.key.value

        limit = self.limit

        min_base = self.min_base

        state = self.state.value

        mailbox = self.mailbox

        not_reported: list[str] | Unset = UNSET
        if not isinstance(self.not_reported, Unset):
            not_reported = self.not_reported



        reported: list[str] | Unset = UNSET
        if not isinstance(self.reported, Unset):
            reported = self.reported



        seeds: dict[str, Any] | Unset = UNSET
        if not isinstance(self.seeds, Unset):
            seeds = self.seeds.to_dict()

        value = self.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "base": base,
            "events": events,
            "key": key,
            "limit": limit,
            "minBase": min_base,
            "state": state,
        })
        if mailbox is not UNSET:
            field_dict["mailbox"] = mailbox
        if not_reported is not UNSET:
            field_dict["notReported"] = not_reported
        if reported is not UNSET:
            field_dict["reported"] = reported
        if seeds is not UNSET:
            field_dict["seeds"] = seeds
        if value is not UNSET:
            field_dict["value"] = value

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.seed_placement import SeedPlacement # noqa: PLC0415
        d = dict(src_dict)
        base = d.pop("base")

        events = d.pop("events")

        key = CanaryGateKey(d.pop("key"))




        limit = d.pop("limit")

        min_base = d.pop("minBase")

        state = CanaryGateState(d.pop("state"))




        mailbox = d.pop("mailbox", UNSET)

        not_reported = cast(list[str], d.pop("notReported", UNSET))


        reported = cast(list[str], d.pop("reported", UNSET))


        _seeds = d.pop("seeds", UNSET)
        seeds: SeedPlacement | Unset
        if isinstance(_seeds,  Unset):
            seeds = UNSET
        else:
            seeds = SeedPlacement.from_dict(_seeds)




        value = d.pop("value", UNSET)

        canary_gate = cls(
            base=base,
            events=events,
            key=key,
            limit=limit,
            min_base=min_base,
            state=state,
            mailbox=mailbox,
            not_reported=not_reported,
            reported=reported,
            seeds=seeds,
            value=value,
        )

        return canary_gate

