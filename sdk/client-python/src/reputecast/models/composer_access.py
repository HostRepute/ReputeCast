from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="ComposerAccess")



@_attrs_define
class ComposerAccess:
    """ 
        Attributes:
            can_edit (bool):
            can_edit_templates (bool):
            can_manage_routing (bool):
            can_send (bool):
            can_send_test_to_others (bool):
            can_view_contacts (bool):
            editable (bool):
            send_helpers (list[str]): Members who hold campaigns:send
            edit_helpers (list[str] | Unset): Members who hold campaigns:edit (when the caller can't edit)
            read_only_reason (str | Unset):
     """

    can_edit: bool
    can_edit_templates: bool
    can_manage_routing: bool
    can_send: bool
    can_send_test_to_others: bool
    can_view_contacts: bool
    editable: bool
    send_helpers: list[str]
    edit_helpers: list[str] | Unset = UNSET
    read_only_reason: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        can_edit = self.can_edit

        can_edit_templates = self.can_edit_templates

        can_manage_routing = self.can_manage_routing

        can_send = self.can_send

        can_send_test_to_others = self.can_send_test_to_others

        can_view_contacts = self.can_view_contacts

        editable = self.editable

        send_helpers = self.send_helpers



        edit_helpers: list[str] | Unset = UNSET
        if not isinstance(self.edit_helpers, Unset):
            edit_helpers = self.edit_helpers



        read_only_reason = self.read_only_reason


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canEdit": can_edit,
            "canEditTemplates": can_edit_templates,
            "canManageRouting": can_manage_routing,
            "canSend": can_send,
            "canSendTestToOthers": can_send_test_to_others,
            "canViewContacts": can_view_contacts,
            "editable": editable,
            "sendHelpers": send_helpers,
        })
        if edit_helpers is not UNSET:
            field_dict["editHelpers"] = edit_helpers
        if read_only_reason is not UNSET:
            field_dict["readOnlyReason"] = read_only_reason

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        can_edit = d.pop("canEdit")

        can_edit_templates = d.pop("canEditTemplates")

        can_manage_routing = d.pop("canManageRouting")

        can_send = d.pop("canSend")

        can_send_test_to_others = d.pop("canSendTestToOthers")

        can_view_contacts = d.pop("canViewContacts")

        editable = d.pop("editable")

        send_helpers = cast(list[str], d.pop("sendHelpers"))


        edit_helpers = cast(list[str], d.pop("editHelpers", UNSET))


        read_only_reason = d.pop("readOnlyReason", UNSET)

        composer_access = cls(
            can_edit=can_edit,
            can_edit_templates=can_edit_templates,
            can_manage_routing=can_manage_routing,
            can_send=can_send,
            can_send_test_to_others=can_send_test_to_others,
            can_view_contacts=can_view_contacts,
            editable=editable,
            send_helpers=send_helpers,
            edit_helpers=edit_helpers,
            read_only_reason=read_only_reason,
        )

        return composer_access

