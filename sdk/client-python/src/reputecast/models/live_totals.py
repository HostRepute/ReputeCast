from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="LiveTotals")



@_attrs_define
class LiveTotals:
    """ 
        Attributes:
            block_bounced (int):
            bounced (int):
            clicked (int):
            complained (int):
            complaint_base (int):
            deferred (int):
            delivered (int):
            hard_bounced (int):
            not_reported (list[str]):
            opened (int):
            opened_human (int):
            opened_machine (int):
            rejected (int):
            soft_bounced (int):
            unknown_bounced (int):
            unsubscribed (int):
     """

    block_bounced: int
    bounced: int
    clicked: int
    complained: int
    complaint_base: int
    deferred: int
    delivered: int
    hard_bounced: int
    not_reported: list[str]
    opened: int
    opened_human: int
    opened_machine: int
    rejected: int
    soft_bounced: int
    unknown_bounced: int
    unsubscribed: int





    def to_dict(self) -> dict[str, Any]:
        block_bounced = self.block_bounced

        bounced = self.bounced

        clicked = self.clicked

        complained = self.complained

        complaint_base = self.complaint_base

        deferred = self.deferred

        delivered = self.delivered

        hard_bounced = self.hard_bounced

        not_reported = self.not_reported



        opened = self.opened

        opened_human = self.opened_human

        opened_machine = self.opened_machine

        rejected = self.rejected

        soft_bounced = self.soft_bounced

        unknown_bounced = self.unknown_bounced

        unsubscribed = self.unsubscribed


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "blockBounced": block_bounced,
            "bounced": bounced,
            "clicked": clicked,
            "complained": complained,
            "complaintBase": complaint_base,
            "deferred": deferred,
            "delivered": delivered,
            "hardBounced": hard_bounced,
            "notReported": not_reported,
            "opened": opened,
            "openedHuman": opened_human,
            "openedMachine": opened_machine,
            "rejected": rejected,
            "softBounced": soft_bounced,
            "unknownBounced": unknown_bounced,
            "unsubscribed": unsubscribed,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        block_bounced = d.pop("blockBounced")

        bounced = d.pop("bounced")

        clicked = d.pop("clicked")

        complained = d.pop("complained")

        complaint_base = d.pop("complaintBase")

        deferred = d.pop("deferred")

        delivered = d.pop("delivered")

        hard_bounced = d.pop("hardBounced")

        not_reported = cast(list[str], d.pop("notReported"))


        opened = d.pop("opened")

        opened_human = d.pop("openedHuman")

        opened_machine = d.pop("openedMachine")

        rejected = d.pop("rejected")

        soft_bounced = d.pop("softBounced")

        unknown_bounced = d.pop("unknownBounced")

        unsubscribed = d.pop("unsubscribed")

        live_totals = cls(
            block_bounced=block_bounced,
            bounced=bounced,
            clicked=clicked,
            complained=complained,
            complaint_base=complaint_base,
            deferred=deferred,
            delivered=delivered,
            hard_bounced=hard_bounced,
            not_reported=not_reported,
            opened=opened,
            opened_human=opened_human,
            opened_machine=opened_machine,
            rejected=rejected,
            soft_bounced=soft_bounced,
            unknown_bounced=unknown_bounced,
            unsubscribed=unsubscribed,
        )

        return live_totals

