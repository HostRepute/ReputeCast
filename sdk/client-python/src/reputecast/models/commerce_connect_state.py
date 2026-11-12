from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.commerce_connect_state_error import CommerceConnectStateError
from ..models.commerce_connect_state_step import CommerceConnectStateStep
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.commerce_list_option import CommerceListOption
  from ..models.commerce_plugin_state import CommercePluginState
  from ..models.commerce_store_summary import CommerceStoreSummary
  from ..models.commerce_sync_choice import CommerceSyncChoice





T = TypeVar("T", bound="CommerceConnectState")



@_attrs_define
class CommerceConnectState:
    """ 
        Attributes:
            app_url (str):
            demo (bool):
            error (CommerceConnectStateError):
            granted (list[str]):
            lists (list[CommerceListOption]):
            months (int):
            new_list_name (str):
            optional_scope (str):
            reachable (bool):
            redirect_url (str):
            scopes (list[str]):
            step (CommerceConnectStateStep):
            store (CommerceStoreSummary):
            syncs (CommerceSyncChoice):
            webhooks (int):
            write_granted (bool):
            client_id_hint (str | Unset):
            installed_at (datetime.datetime | Unset):
            key_hint (str | Unset):
            plugin (CommercePluginState | Unset):
            reachable_checked_at (datetime.datetime | Unset):
     """

    app_url: str
    demo: bool
    error: CommerceConnectStateError
    granted: list[str]
    lists: list[CommerceListOption]
    months: int
    new_list_name: str
    optional_scope: str
    reachable: bool
    redirect_url: str
    scopes: list[str]
    step: CommerceConnectStateStep
    store: CommerceStoreSummary
    syncs: CommerceSyncChoice
    webhooks: int
    write_granted: bool
    client_id_hint: str | Unset = UNSET
    installed_at: datetime.datetime | Unset = UNSET
    key_hint: str | Unset = UNSET
    plugin: CommercePluginState | Unset = UNSET
    reachable_checked_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.commerce_list_option import CommerceListOption # noqa: PLC0415
        from ..models.commerce_plugin_state import CommercePluginState # noqa: PLC0415
        from ..models.commerce_store_summary import CommerceStoreSummary # noqa: PLC0415
        from ..models.commerce_sync_choice import CommerceSyncChoice # noqa: PLC0415
        app_url = self.app_url

        demo = self.demo

        error = self.error.value

        granted = self.granted



        lists = []
        for lists_item_data in self.lists:
            lists_item = lists_item_data.to_dict()
            lists.append(lists_item)



        months = self.months

        new_list_name = self.new_list_name

        optional_scope = self.optional_scope

        reachable = self.reachable

        redirect_url = self.redirect_url

        scopes = self.scopes



        step = self.step.value

        store = self.store.to_dict()

        syncs = self.syncs.to_dict()

        webhooks = self.webhooks

        write_granted = self.write_granted

        client_id_hint = self.client_id_hint

        installed_at: str | Unset = UNSET
        if not isinstance(self.installed_at, Unset):
            installed_at = self.installed_at.isoformat()

        key_hint = self.key_hint

        plugin: dict[str, Any] | Unset = UNSET
        if not isinstance(self.plugin, Unset):
            plugin = self.plugin.to_dict()

        reachable_checked_at: str | Unset = UNSET
        if not isinstance(self.reachable_checked_at, Unset):
            reachable_checked_at = self.reachable_checked_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "appUrl": app_url,
            "demo": demo,
            "error": error,
            "granted": granted,
            "lists": lists,
            "months": months,
            "newListName": new_list_name,
            "optionalScope": optional_scope,
            "reachable": reachable,
            "redirectUrl": redirect_url,
            "scopes": scopes,
            "step": step,
            "store": store,
            "syncs": syncs,
            "webhooks": webhooks,
            "writeGranted": write_granted,
        })
        if client_id_hint is not UNSET:
            field_dict["clientIdHint"] = client_id_hint
        if installed_at is not UNSET:
            field_dict["installedAt"] = installed_at
        if key_hint is not UNSET:
            field_dict["keyHint"] = key_hint
        if plugin is not UNSET:
            field_dict["plugin"] = plugin
        if reachable_checked_at is not UNSET:
            field_dict["reachableCheckedAt"] = reachable_checked_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.commerce_list_option import CommerceListOption # noqa: PLC0415
        from ..models.commerce_plugin_state import CommercePluginState # noqa: PLC0415
        from ..models.commerce_store_summary import CommerceStoreSummary # noqa: PLC0415
        from ..models.commerce_sync_choice import CommerceSyncChoice # noqa: PLC0415
        d = dict(src_dict)
        app_url = d.pop("appUrl")

        demo = d.pop("demo")

        error = CommerceConnectStateError(d.pop("error"))




        granted = cast(list[str], d.pop("granted"))


        lists = []
        _lists = d.pop("lists")
        for lists_item_data in (_lists):
            lists_item = CommerceListOption.from_dict(lists_item_data)



            lists.append(lists_item)


        months = d.pop("months")

        new_list_name = d.pop("newListName")

        optional_scope = d.pop("optionalScope")

        reachable = d.pop("reachable")

        redirect_url = d.pop("redirectUrl")

        scopes = cast(list[str], d.pop("scopes"))


        step = CommerceConnectStateStep(d.pop("step"))




        store = CommerceStoreSummary.from_dict(d.pop("store"))




        syncs = CommerceSyncChoice.from_dict(d.pop("syncs"))




        webhooks = d.pop("webhooks")

        write_granted = d.pop("writeGranted")

        client_id_hint = d.pop("clientIdHint", UNSET)

        _installed_at = d.pop("installedAt", UNSET)
        installed_at: datetime.datetime | Unset
        if isinstance(_installed_at,  Unset):
            installed_at = UNSET
        else:
            installed_at = datetime.datetime.fromisoformat(_installed_at)




        key_hint = d.pop("keyHint", UNSET)

        _plugin = d.pop("plugin", UNSET)
        plugin: CommercePluginState | Unset
        if isinstance(_plugin,  Unset):
            plugin = UNSET
        else:
            plugin = CommercePluginState.from_dict(_plugin)




        _reachable_checked_at = d.pop("reachableCheckedAt", UNSET)
        reachable_checked_at: datetime.datetime | Unset
        if isinstance(_reachable_checked_at,  Unset):
            reachable_checked_at = UNSET
        else:
            reachable_checked_at = datetime.datetime.fromisoformat(_reachable_checked_at)




        commerce_connect_state = cls(
            app_url=app_url,
            demo=demo,
            error=error,
            granted=granted,
            lists=lists,
            months=months,
            new_list_name=new_list_name,
            optional_scope=optional_scope,
            reachable=reachable,
            redirect_url=redirect_url,
            scopes=scopes,
            step=step,
            store=store,
            syncs=syncs,
            webhooks=webhooks,
            write_granted=write_granted,
            client_id_hint=client_id_hint,
            installed_at=installed_at,
            key_hint=key_hint,
            plugin=plugin,
            reachable_checked_at=reachable_checked_at,
        )

        return commerce_connect_state

