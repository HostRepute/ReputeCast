from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.seed_mailbox_result import SeedMailboxResult





T = TypeVar("T", bound="SeedPlacement")



@_attrs_define
class SeedPlacement:
    """ 
        Attributes:
            by_mailbox (list[SeedMailboxResult]):
            inbox (int):
            missing (int):
            pending (int):
            promotions (int):
            spam (int):
     """

    by_mailbox: list[SeedMailboxResult]
    inbox: int
    missing: int
    pending: int
    promotions: int
    spam: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.seed_mailbox_result import SeedMailboxResult # noqa: PLC0415
        by_mailbox = []
        for by_mailbox_item_data in self.by_mailbox:
            by_mailbox_item = by_mailbox_item_data.to_dict()
            by_mailbox.append(by_mailbox_item)



        inbox = self.inbox

        missing = self.missing

        pending = self.pending

        promotions = self.promotions

        spam = self.spam


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "byMailbox": by_mailbox,
            "inbox": inbox,
            "missing": missing,
            "pending": pending,
            "promotions": promotions,
            "spam": spam,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.seed_mailbox_result import SeedMailboxResult # noqa: PLC0415
        d = dict(src_dict)
        by_mailbox = []
        _by_mailbox = d.pop("byMailbox")
        for by_mailbox_item_data in (_by_mailbox):
            by_mailbox_item = SeedMailboxResult.from_dict(by_mailbox_item_data)



            by_mailbox.append(by_mailbox_item)


        inbox = d.pop("inbox")

        missing = d.pop("missing")

        pending = d.pop("pending")

        promotions = d.pop("promotions")

        spam = d.pop("spam")

        seed_placement = cls(
            by_mailbox=by_mailbox,
            inbox=inbox,
            missing=missing,
            pending=pending,
            promotions=promotions,
            spam=spam,
        )

        return seed_placement

