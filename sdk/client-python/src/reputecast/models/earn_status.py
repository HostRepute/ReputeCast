from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.earn_status_mode import EarnStatusMode
from ..types import UNSET, Unset






T = TypeVar("T", bound="EarnStatus")



@_attrs_define
class EarnStatus:
    """ 
        Attributes:
            can_manage (bool):
            mode (EarnStatusMode):
            test_only (bool):
            upgrade_url (str):
            account (str | Unset):
            health (str | Unset):
            provider (str | Unset):
            provider_name (str | Unset):
            setup (str | Unset):
     """

    can_manage: bool
    mode: EarnStatusMode
    test_only: bool
    upgrade_url: str
    account: str | Unset = UNSET
    health: str | Unset = UNSET
    provider: str | Unset = UNSET
    provider_name: str | Unset = UNSET
    setup: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        can_manage = self.can_manage

        mode = self.mode.value

        test_only = self.test_only

        upgrade_url = self.upgrade_url

        account = self.account

        health = self.health

        provider = self.provider

        provider_name = self.provider_name

        setup = self.setup


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canManage": can_manage,
            "mode": mode,
            "testOnly": test_only,
            "upgradeUrl": upgrade_url,
        })
        if account is not UNSET:
            field_dict["account"] = account
        if health is not UNSET:
            field_dict["health"] = health
        if provider is not UNSET:
            field_dict["provider"] = provider
        if provider_name is not UNSET:
            field_dict["providerName"] = provider_name
        if setup is not UNSET:
            field_dict["setup"] = setup

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        can_manage = d.pop("canManage")

        mode = EarnStatusMode(d.pop("mode"))




        test_only = d.pop("testOnly")

        upgrade_url = d.pop("upgradeUrl")

        account = d.pop("account", UNSET)

        health = d.pop("health", UNSET)

        provider = d.pop("provider", UNSET)

        provider_name = d.pop("providerName", UNSET)

        setup = d.pop("setup", UNSET)

        earn_status = cls(
            can_manage=can_manage,
            mode=mode,
            test_only=test_only,
            upgrade_url=upgrade_url,
            account=account,
            health=health,
            provider=provider,
            provider_name=provider_name,
            setup=setup,
        )

        return earn_status

