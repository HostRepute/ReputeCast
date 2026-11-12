from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.sending_defaults import SendingDefaults
  from ..models.settings_provider_card import SettingsProviderCard





T = TypeVar("T", bound="SettingsProviders")



@_attrs_define
class SettingsProviders:
    """ 
        Attributes:
            can_manage (bool):
            defaults (SendingDefaults):
            helpers (list[str]):
            month_start (datetime.datetime):
            providers (list[SettingsProviderCard]):
     """

    can_manage: bool
    defaults: SendingDefaults
    helpers: list[str]
    month_start: datetime.datetime
    providers: list[SettingsProviderCard]





    def to_dict(self) -> dict[str, Any]:
        from ..models.sending_defaults import SendingDefaults # noqa: PLC0415
        from ..models.settings_provider_card import SettingsProviderCard # noqa: PLC0415
        can_manage = self.can_manage

        defaults = self.defaults.to_dict()

        helpers = self.helpers



        month_start = self.month_start.isoformat()

        providers = []
        for providers_item_data in self.providers:
            providers_item = providers_item_data.to_dict()
            providers.append(providers_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canManage": can_manage,
            "defaults": defaults,
            "helpers": helpers,
            "monthStart": month_start,
            "providers": providers,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.sending_defaults import SendingDefaults # noqa: PLC0415
        from ..models.settings_provider_card import SettingsProviderCard # noqa: PLC0415
        d = dict(src_dict)
        can_manage = d.pop("canManage")

        defaults = SendingDefaults.from_dict(d.pop("defaults"))




        helpers = cast(list[str], d.pop("helpers"))


        month_start = datetime.datetime.fromisoformat(d.pop("monthStart"))




        providers = []
        _providers = d.pop("providers")
        for providers_item_data in (_providers):
            providers_item = SettingsProviderCard.from_dict(providers_item_data)



            providers.append(providers_item)


        settings_providers = cls(
            can_manage=can_manage,
            defaults=defaults,
            helpers=helpers,
            month_start=month_start,
            providers=providers,
        )

        return settings_providers

