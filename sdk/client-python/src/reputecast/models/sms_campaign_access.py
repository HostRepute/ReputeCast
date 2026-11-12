from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="SmsCampaignAccess")



@_attrs_define
class SmsCampaignAccess:
    """ 
        Attributes:
            can_edit (bool):
            can_manage_rules (bool):
            can_send (bool):
            can_view_contacts (bool):
            helpers (list[str]):
            missing (list[str]):
     """

    can_edit: bool
    can_manage_rules: bool
    can_send: bool
    can_view_contacts: bool
    helpers: list[str]
    missing: list[str]





    def to_dict(self) -> dict[str, Any]:
        can_edit = self.can_edit

        can_manage_rules = self.can_manage_rules

        can_send = self.can_send

        can_view_contacts = self.can_view_contacts

        helpers = self.helpers



        missing = self.missing




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canEdit": can_edit,
            "canManageRules": can_manage_rules,
            "canSend": can_send,
            "canViewContacts": can_view_contacts,
            "helpers": helpers,
            "missing": missing,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        can_edit = d.pop("canEdit")

        can_manage_rules = d.pop("canManageRules")

        can_send = d.pop("canSend")

        can_view_contacts = d.pop("canViewContacts")

        helpers = cast(list[str], d.pop("helpers"))


        missing = cast(list[str], d.pop("missing"))


        sms_campaign_access = cls(
            can_edit=can_edit,
            can_manage_rules=can_manage_rules,
            can_send=can_send,
            can_view_contacts=can_view_contacts,
            helpers=helpers,
            missing=missing,
        )

        return sms_campaign_access

