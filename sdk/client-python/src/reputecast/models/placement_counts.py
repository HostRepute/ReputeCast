from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.placement_not_checked import PlacementNotChecked





T = TypeVar("T", bound="PlacementCounts")



@_attrs_define
class PlacementCounts:
    """ 
        Attributes:
            copies (int):
            failed (int):
            found (int):
            inbox (int):
            not_arrived (int):
            not_checked (int):
            not_checked_seeds (list[PlacementNotChecked]):
            spam (int):
            tab (int):
            tabs (list[str]):
            unsent (int):
            waiting (int):
     """

    copies: int
    failed: int
    found: int
    inbox: int
    not_arrived: int
    not_checked: int
    not_checked_seeds: list[PlacementNotChecked]
    spam: int
    tab: int
    tabs: list[str]
    unsent: int
    waiting: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.placement_not_checked import PlacementNotChecked # noqa: PLC0415
        copies = self.copies

        failed = self.failed

        found = self.found

        inbox = self.inbox

        not_arrived = self.not_arrived

        not_checked = self.not_checked

        not_checked_seeds = []
        for not_checked_seeds_item_data in self.not_checked_seeds:
            not_checked_seeds_item = not_checked_seeds_item_data.to_dict()
            not_checked_seeds.append(not_checked_seeds_item)



        spam = self.spam

        tab = self.tab

        tabs = self.tabs



        unsent = self.unsent

        waiting = self.waiting


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "copies": copies,
            "failed": failed,
            "found": found,
            "inbox": inbox,
            "notArrived": not_arrived,
            "notChecked": not_checked,
            "notCheckedSeeds": not_checked_seeds,
            "spam": spam,
            "tab": tab,
            "tabs": tabs,
            "unsent": unsent,
            "waiting": waiting,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.placement_not_checked import PlacementNotChecked # noqa: PLC0415
        d = dict(src_dict)
        copies = d.pop("copies")

        failed = d.pop("failed")

        found = d.pop("found")

        inbox = d.pop("inbox")

        not_arrived = d.pop("notArrived")

        not_checked = d.pop("notChecked")

        not_checked_seeds = []
        _not_checked_seeds = d.pop("notCheckedSeeds")
        for not_checked_seeds_item_data in (_not_checked_seeds):
            not_checked_seeds_item = PlacementNotChecked.from_dict(not_checked_seeds_item_data)



            not_checked_seeds.append(not_checked_seeds_item)


        spam = d.pop("spam")

        tab = d.pop("tab")

        tabs = cast(list[str], d.pop("tabs"))


        unsent = d.pop("unsent")

        waiting = d.pop("waiting")

        placement_counts = cls(
            copies=copies,
            failed=failed,
            found=found,
            inbox=inbox,
            not_arrived=not_arrived,
            not_checked=not_checked,
            not_checked_seeds=not_checked_seeds,
            spam=spam,
            tab=tab,
            tabs=tabs,
            unsent=unsent,
            waiting=waiting,
        )

        return placement_counts

