from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="TransactionalAccess")



@_attrs_define
class TransactionalAccess:
    """ 
        Attributes:
            content (bool):
            edit (bool):
            edit_route (bool):
            manage_keys (bool):
            publish (bool):
            send (bool):
     """

    content: bool
    edit: bool
    edit_route: bool
    manage_keys: bool
    publish: bool
    send: bool





    def to_dict(self) -> dict[str, Any]:
        content = self.content

        edit = self.edit

        edit_route = self.edit_route

        manage_keys = self.manage_keys

        publish = self.publish

        send = self.send


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "content": content,
            "edit": edit,
            "editRoute": edit_route,
            "manageKeys": manage_keys,
            "publish": publish,
            "send": send,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        content = d.pop("content")

        edit = d.pop("edit")

        edit_route = d.pop("editRoute")

        manage_keys = d.pop("manageKeys")

        publish = d.pop("publish")

        send = d.pop("send")

        transactional_access = cls(
            content=content,
            edit=edit,
            edit_route=edit_route,
            manage_keys=manage_keys,
            publish=publish,
            send=send,
        )

        return transactional_access

