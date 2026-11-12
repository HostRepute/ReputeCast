from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.sender_input_default_for_item import SenderInputDefaultForItem
from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="SenderInput")



@_attrs_define
class SenderInput:
    """ 
        Attributes:
            name (str):
            default_for (list[SenderInputDefaultForItem] | Unset): transactional needs a sender on a Transactional domain
            email (str | Unset): Required when creating; the address can't change later
            not_used_with (list[str] | Unset):
            note (str | Unset): Shown when the sender is no stream's default, e.g. personal sends
            reply_to (str | Unset):
     """

    name: str
    default_for: list[SenderInputDefaultForItem] | Unset = UNSET
    email: str | Unset = UNSET
    not_used_with: list[str] | Unset = UNSET
    note: str | Unset = UNSET
    reply_to: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        name = self.name

        default_for: list[str] | Unset = UNSET
        if not isinstance(self.default_for, Unset):
            default_for = []
            for default_for_item_data in self.default_for:
                default_for_item = default_for_item_data.value
                default_for.append(default_for_item)



        email = self.email

        not_used_with: list[str] | Unset = UNSET
        if not isinstance(self.not_used_with, Unset):
            not_used_with = self.not_used_with



        note = self.note

        reply_to = self.reply_to


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "name": name,
        })
        if default_for is not UNSET:
            field_dict["defaultFor"] = default_for
        if email is not UNSET:
            field_dict["email"] = email
        if not_used_with is not UNSET:
            field_dict["notUsedWith"] = not_used_with
        if note is not UNSET:
            field_dict["note"] = note
        if reply_to is not UNSET:
            field_dict["replyTo"] = reply_to

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        name = d.pop("name")

        _default_for = d.pop("defaultFor", UNSET)
        default_for: list[SenderInputDefaultForItem] | Unset = UNSET
        if _default_for is not UNSET:
            default_for = []
            for default_for_item_data in _default_for:
                default_for_item = SenderInputDefaultForItem(default_for_item_data)



                default_for.append(default_for_item)


        email = d.pop("email", UNSET)

        not_used_with = cast(list[str], d.pop("notUsedWith", UNSET))


        note = d.pop("note", UNSET)

        reply_to = d.pop("replyTo", UNSET)

        sender_input = cls(
            name=name,
            default_for=default_for,
            email=email,
            not_used_with=not_used_with,
            note=note,
            reply_to=reply_to,
        )

        return sender_input

