from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="PushStepEstimate")



@_attrs_define
class PushStepEstimate:
    """ 
        Attributes:
            contacts (int):
            linked (int):
            push_on (bool):
            quiet (bool):
            subscriptions (int):
            window (str):
     """

    contacts: int
    linked: int
    push_on: bool
    quiet: bool
    subscriptions: int
    window: str





    def to_dict(self) -> dict[str, Any]:
        contacts = self.contacts

        linked = self.linked

        push_on = self.push_on

        quiet = self.quiet

        subscriptions = self.subscriptions

        window = self.window


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "contacts": contacts,
            "linked": linked,
            "pushOn": push_on,
            "quiet": quiet,
            "subscriptions": subscriptions,
            "window": window,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        contacts = d.pop("contacts")

        linked = d.pop("linked")

        push_on = d.pop("pushOn")

        quiet = d.pop("quiet")

        subscriptions = d.pop("subscriptions")

        window = d.pop("window")

        push_step_estimate = cls(
            contacts=contacts,
            linked=linked,
            push_on=push_on,
            quiet=quiet,
            subscriptions=subscriptions,
            window=window,
        )

        return push_step_estimate

