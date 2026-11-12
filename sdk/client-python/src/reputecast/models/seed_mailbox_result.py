from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.seed_mailbox_result_mailbox import SeedMailboxResultMailbox






T = TypeVar("T", bound="SeedMailboxResult")



@_attrs_define
class SeedMailboxResult:
    """ 
        Attributes:
            inbox (int):
            mailbox (SeedMailboxResultMailbox):
            promotions (int):
            seeds (int):
            spam (int):
     """

    inbox: int
    mailbox: SeedMailboxResultMailbox
    promotions: int
    seeds: int
    spam: int





    def to_dict(self) -> dict[str, Any]:
        inbox = self.inbox

        mailbox = self.mailbox.value

        promotions = self.promotions

        seeds = self.seeds

        spam = self.spam


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "inbox": inbox,
            "mailbox": mailbox,
            "promotions": promotions,
            "seeds": seeds,
            "spam": spam,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        inbox = d.pop("inbox")

        mailbox = SeedMailboxResultMailbox(d.pop("mailbox"))




        promotions = d.pop("promotions")

        seeds = d.pop("seeds")

        spam = d.pop("spam")

        seed_mailbox_result = cls(
            inbox=inbox,
            mailbox=mailbox,
            promotions=promotions,
            seeds=seeds,
            spam=spam,
        )

        return seed_mailbox_result

