from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.commerce_activity_item import CommerceActivityItem
  from ..models.commerce_attribution import CommerceAttribution
  from ..models.commerce_consent import CommerceConsent
  from ..models.commerce_flow import CommerceFlow
  from ..models.commerce_list_option import CommerceListOption
  from ..models.commerce_numbers import CommerceNumbers
  from ..models.commerce_plugin_state import CommercePluginState
  from ..models.commerce_recovered_orders import CommerceRecoveredOrders
  from ..models.commerce_store_summary import CommerceStoreSummary
  from ..models.commerce_sync_facts import CommerceSyncFacts
  from ..models.commerce_use import CommerceUse
  from ..models.commerce_webhook import CommerceWebhook





T = TypeVar("T", bound="CommerceStorePage")



@_attrs_define
class CommerceStorePage:
    """ 
        Attributes:
            activity (list[CommerceActivityItem]):
            api_version (str):
            app_name (str):
            attribution (CommerceAttribution):
            can_manage (bool):
            capabilities (list[str]):
            connected_by (str):
            consent (CommerceConsent):
            flows (list[CommerceFlow]):
            helpers (list[str]):
            lists (list[CommerceListOption]):
            numbers (CommerceNumbers):
            public_domain (str):
            recovered (CommerceRecoveredOrders):
            store (CommerceStoreSummary):
            sync (CommerceSyncFacts):
            used_by (list[CommerceUse]):
            webhooks (list[CommerceWebhook]):
            connected_at (datetime.datetime | Unset):
            plugin (CommercePluginState | Unset):
     """

    activity: list[CommerceActivityItem]
    api_version: str
    app_name: str
    attribution: CommerceAttribution
    can_manage: bool
    capabilities: list[str]
    connected_by: str
    consent: CommerceConsent
    flows: list[CommerceFlow]
    helpers: list[str]
    lists: list[CommerceListOption]
    numbers: CommerceNumbers
    public_domain: str
    recovered: CommerceRecoveredOrders
    store: CommerceStoreSummary
    sync: CommerceSyncFacts
    used_by: list[CommerceUse]
    webhooks: list[CommerceWebhook]
    connected_at: datetime.datetime | Unset = UNSET
    plugin: CommercePluginState | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.commerce_activity_item import CommerceActivityItem # noqa: PLC0415
        from ..models.commerce_attribution import CommerceAttribution # noqa: PLC0415
        from ..models.commerce_consent import CommerceConsent # noqa: PLC0415
        from ..models.commerce_flow import CommerceFlow # noqa: PLC0415
        from ..models.commerce_list_option import CommerceListOption # noqa: PLC0415
        from ..models.commerce_numbers import CommerceNumbers # noqa: PLC0415
        from ..models.commerce_plugin_state import CommercePluginState # noqa: PLC0415
        from ..models.commerce_recovered_orders import CommerceRecoveredOrders # noqa: PLC0415
        from ..models.commerce_store_summary import CommerceStoreSummary # noqa: PLC0415
        from ..models.commerce_sync_facts import CommerceSyncFacts # noqa: PLC0415
        from ..models.commerce_use import CommerceUse # noqa: PLC0415
        from ..models.commerce_webhook import CommerceWebhook # noqa: PLC0415
        activity = []
        for activity_item_data in self.activity:
            activity_item = activity_item_data.to_dict()
            activity.append(activity_item)



        api_version = self.api_version

        app_name = self.app_name

        attribution = self.attribution.to_dict()

        can_manage = self.can_manage

        capabilities = self.capabilities



        connected_by = self.connected_by

        consent = self.consent.to_dict()

        flows = []
        for flows_item_data in self.flows:
            flows_item = flows_item_data.to_dict()
            flows.append(flows_item)



        helpers = self.helpers



        lists = []
        for lists_item_data in self.lists:
            lists_item = lists_item_data.to_dict()
            lists.append(lists_item)



        numbers = self.numbers.to_dict()

        public_domain = self.public_domain

        recovered = self.recovered.to_dict()

        store = self.store.to_dict()

        sync = self.sync.to_dict()

        used_by = []
        for used_by_item_data in self.used_by:
            used_by_item = used_by_item_data.to_dict()
            used_by.append(used_by_item)



        webhooks = []
        for webhooks_item_data in self.webhooks:
            webhooks_item = webhooks_item_data.to_dict()
            webhooks.append(webhooks_item)



        connected_at: str | Unset = UNSET
        if not isinstance(self.connected_at, Unset):
            connected_at = self.connected_at.isoformat()

        plugin: dict[str, Any] | Unset = UNSET
        if not isinstance(self.plugin, Unset):
            plugin = self.plugin.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "activity": activity,
            "apiVersion": api_version,
            "appName": app_name,
            "attribution": attribution,
            "canManage": can_manage,
            "capabilities": capabilities,
            "connectedBy": connected_by,
            "consent": consent,
            "flows": flows,
            "helpers": helpers,
            "lists": lists,
            "numbers": numbers,
            "publicDomain": public_domain,
            "recovered": recovered,
            "store": store,
            "sync": sync,
            "usedBy": used_by,
            "webhooks": webhooks,
        })
        if connected_at is not UNSET:
            field_dict["connectedAt"] = connected_at
        if plugin is not UNSET:
            field_dict["plugin"] = plugin

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.commerce_activity_item import CommerceActivityItem # noqa: PLC0415
        from ..models.commerce_attribution import CommerceAttribution # noqa: PLC0415
        from ..models.commerce_consent import CommerceConsent # noqa: PLC0415
        from ..models.commerce_flow import CommerceFlow # noqa: PLC0415
        from ..models.commerce_list_option import CommerceListOption # noqa: PLC0415
        from ..models.commerce_numbers import CommerceNumbers # noqa: PLC0415
        from ..models.commerce_plugin_state import CommercePluginState # noqa: PLC0415
        from ..models.commerce_recovered_orders import CommerceRecoveredOrders # noqa: PLC0415
        from ..models.commerce_store_summary import CommerceStoreSummary # noqa: PLC0415
        from ..models.commerce_sync_facts import CommerceSyncFacts # noqa: PLC0415
        from ..models.commerce_use import CommerceUse # noqa: PLC0415
        from ..models.commerce_webhook import CommerceWebhook # noqa: PLC0415
        d = dict(src_dict)
        activity = []
        _activity = d.pop("activity")
        for activity_item_data in (_activity):
            activity_item = CommerceActivityItem.from_dict(activity_item_data)



            activity.append(activity_item)


        api_version = d.pop("apiVersion")

        app_name = d.pop("appName")

        attribution = CommerceAttribution.from_dict(d.pop("attribution"))




        can_manage = d.pop("canManage")

        capabilities = cast(list[str], d.pop("capabilities"))


        connected_by = d.pop("connectedBy")

        consent = CommerceConsent.from_dict(d.pop("consent"))




        flows = []
        _flows = d.pop("flows")
        for flows_item_data in (_flows):
            flows_item = CommerceFlow.from_dict(flows_item_data)



            flows.append(flows_item)


        helpers = cast(list[str], d.pop("helpers"))


        lists = []
        _lists = d.pop("lists")
        for lists_item_data in (_lists):
            lists_item = CommerceListOption.from_dict(lists_item_data)



            lists.append(lists_item)


        numbers = CommerceNumbers.from_dict(d.pop("numbers"))




        public_domain = d.pop("publicDomain")

        recovered = CommerceRecoveredOrders.from_dict(d.pop("recovered"))




        store = CommerceStoreSummary.from_dict(d.pop("store"))




        sync = CommerceSyncFacts.from_dict(d.pop("sync"))




        used_by = []
        _used_by = d.pop("usedBy")
        for used_by_item_data in (_used_by):
            used_by_item = CommerceUse.from_dict(used_by_item_data)



            used_by.append(used_by_item)


        webhooks = []
        _webhooks = d.pop("webhooks")
        for webhooks_item_data in (_webhooks):
            webhooks_item = CommerceWebhook.from_dict(webhooks_item_data)



            webhooks.append(webhooks_item)


        _connected_at = d.pop("connectedAt", UNSET)
        connected_at: datetime.datetime | Unset
        if isinstance(_connected_at,  Unset):
            connected_at = UNSET
        else:
            connected_at = datetime.datetime.fromisoformat(_connected_at)




        _plugin = d.pop("plugin", UNSET)
        plugin: CommercePluginState | Unset
        if isinstance(_plugin,  Unset):
            plugin = UNSET
        else:
            plugin = CommercePluginState.from_dict(_plugin)




        commerce_store_page = cls(
            activity=activity,
            api_version=api_version,
            app_name=app_name,
            attribution=attribution,
            can_manage=can_manage,
            capabilities=capabilities,
            connected_by=connected_by,
            consent=consent,
            flows=flows,
            helpers=helpers,
            lists=lists,
            numbers=numbers,
            public_domain=public_domain,
            recovered=recovered,
            store=store,
            sync=sync,
            used_by=used_by,
            webhooks=webhooks,
            connected_at=connected_at,
            plugin=plugin,
        )

        return commerce_store_page

