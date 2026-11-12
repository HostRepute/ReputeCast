from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="ReviewAccess")



@_attrs_define
class ReviewAccess:
    """ 
        Attributes:
            can_approve (bool):
            can_edit (bool):
            can_manage_routing (bool):
            can_send (bool):
            can_send_test_to_others (bool):
            edit_helpers (list[str]):
            email (str):
            has_password (bool):
            is_owner (bool):
            send_helpers (list[str]):
     """

    can_approve: bool
    can_edit: bool
    can_manage_routing: bool
    can_send: bool
    can_send_test_to_others: bool
    edit_helpers: list[str]
    email: str
    has_password: bool
    is_owner: bool
    send_helpers: list[str]





    def to_dict(self) -> dict[str, Any]:
        can_approve = self.can_approve

        can_edit = self.can_edit

        can_manage_routing = self.can_manage_routing

        can_send = self.can_send

        can_send_test_to_others = self.can_send_test_to_others

        edit_helpers = self.edit_helpers



        email = self.email

        has_password = self.has_password

        is_owner = self.is_owner

        send_helpers = self.send_helpers




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canApprove": can_approve,
            "canEdit": can_edit,
            "canManageRouting": can_manage_routing,
            "canSend": can_send,
            "canSendTestToOthers": can_send_test_to_others,
            "editHelpers": edit_helpers,
            "email": email,
            "hasPassword": has_password,
            "isOwner": is_owner,
            "sendHelpers": send_helpers,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        can_approve = d.pop("canApprove")

        can_edit = d.pop("canEdit")

        can_manage_routing = d.pop("canManageRouting")

        can_send = d.pop("canSend")

        can_send_test_to_others = d.pop("canSendTestToOthers")

        edit_helpers = cast(list[str], d.pop("editHelpers"))


        email = d.pop("email")

        has_password = d.pop("hasPassword")

        is_owner = d.pop("isOwner")

        send_helpers = cast(list[str], d.pop("sendHelpers"))


        review_access = cls(
            can_approve=can_approve,
            can_edit=can_edit,
            can_manage_routing=can_manage_routing,
            can_send=can_send,
            can_send_test_to_others=can_send_test_to_others,
            edit_helpers=edit_helpers,
            email=email,
            has_password=has_password,
            is_owner=is_owner,
            send_helpers=send_helpers,
        )

        return review_access

