from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.integration_word_press_page_status import IntegrationWordPressPageStatus
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.integration_list_option import IntegrationListOption
  from ..models.integration_option import IntegrationOption
  from ..models.integration_reason import IntegrationReason
  from ..models.integration_site import IntegrationSite





T = TypeVar("T", bound="IntegrationWordPressPage")



@_attrs_define
class IntegrationWordPressPage:
    """ 
        Attributes:
            can_edit_forms (bool):
            can_manage (bool):
            helpers (list[str]):
            instance_url (str):
            latest_plugin (str):
            lists (list[IntegrationListOption]):
            members (list[IntegrationOption]):
            reasons (list[IntegrationReason]):
            senders (list[IntegrationOption]):
            sites (list[IntegrationSite]):
            status (IntegrationWordPressPageStatus):
            templates (list[IntegrationOption]):
            update_available (bool):
            connected_by (str | Unset):
     """

    can_edit_forms: bool
    can_manage: bool
    helpers: list[str]
    instance_url: str
    latest_plugin: str
    lists: list[IntegrationListOption]
    members: list[IntegrationOption]
    reasons: list[IntegrationReason]
    senders: list[IntegrationOption]
    sites: list[IntegrationSite]
    status: IntegrationWordPressPageStatus
    templates: list[IntegrationOption]
    update_available: bool
    connected_by: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.integration_list_option import IntegrationListOption # noqa: PLC0415
        from ..models.integration_option import IntegrationOption # noqa: PLC0415
        from ..models.integration_reason import IntegrationReason # noqa: PLC0415
        from ..models.integration_site import IntegrationSite # noqa: PLC0415
        can_edit_forms = self.can_edit_forms

        can_manage = self.can_manage

        helpers = self.helpers



        instance_url = self.instance_url

        latest_plugin = self.latest_plugin

        lists = []
        for lists_item_data in self.lists:
            lists_item = lists_item_data.to_dict()
            lists.append(lists_item)



        members = []
        for members_item_data in self.members:
            members_item = members_item_data.to_dict()
            members.append(members_item)



        reasons = []
        for reasons_item_data in self.reasons:
            reasons_item = reasons_item_data.to_dict()
            reasons.append(reasons_item)



        senders = []
        for senders_item_data in self.senders:
            senders_item = senders_item_data.to_dict()
            senders.append(senders_item)



        sites = []
        for sites_item_data in self.sites:
            sites_item = sites_item_data.to_dict()
            sites.append(sites_item)



        status = self.status.value

        templates = []
        for templates_item_data in self.templates:
            templates_item = templates_item_data.to_dict()
            templates.append(templates_item)



        update_available = self.update_available

        connected_by = self.connected_by


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canEditForms": can_edit_forms,
            "canManage": can_manage,
            "helpers": helpers,
            "instanceUrl": instance_url,
            "latestPlugin": latest_plugin,
            "lists": lists,
            "members": members,
            "reasons": reasons,
            "senders": senders,
            "sites": sites,
            "status": status,
            "templates": templates,
            "updateAvailable": update_available,
        })
        if connected_by is not UNSET:
            field_dict["connectedBy"] = connected_by

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.integration_list_option import IntegrationListOption # noqa: PLC0415
        from ..models.integration_option import IntegrationOption # noqa: PLC0415
        from ..models.integration_reason import IntegrationReason # noqa: PLC0415
        from ..models.integration_site import IntegrationSite # noqa: PLC0415
        d = dict(src_dict)
        can_edit_forms = d.pop("canEditForms")

        can_manage = d.pop("canManage")

        helpers = cast(list[str], d.pop("helpers"))


        instance_url = d.pop("instanceUrl")

        latest_plugin = d.pop("latestPlugin")

        lists = []
        _lists = d.pop("lists")
        for lists_item_data in (_lists):
            lists_item = IntegrationListOption.from_dict(lists_item_data)



            lists.append(lists_item)


        members = []
        _members = d.pop("members")
        for members_item_data in (_members):
            members_item = IntegrationOption.from_dict(members_item_data)



            members.append(members_item)


        reasons = []
        _reasons = d.pop("reasons")
        for reasons_item_data in (_reasons):
            reasons_item = IntegrationReason.from_dict(reasons_item_data)



            reasons.append(reasons_item)


        senders = []
        _senders = d.pop("senders")
        for senders_item_data in (_senders):
            senders_item = IntegrationOption.from_dict(senders_item_data)



            senders.append(senders_item)


        sites = []
        _sites = d.pop("sites")
        for sites_item_data in (_sites):
            sites_item = IntegrationSite.from_dict(sites_item_data)



            sites.append(sites_item)


        status = IntegrationWordPressPageStatus(d.pop("status"))




        templates = []
        _templates = d.pop("templates")
        for templates_item_data in (_templates):
            templates_item = IntegrationOption.from_dict(templates_item_data)



            templates.append(templates_item)


        update_available = d.pop("updateAvailable")

        connected_by = d.pop("connectedBy", UNSET)

        integration_word_press_page = cls(
            can_edit_forms=can_edit_forms,
            can_manage=can_manage,
            helpers=helpers,
            instance_url=instance_url,
            latest_plugin=latest_plugin,
            lists=lists,
            members=members,
            reasons=reasons,
            senders=senders,
            sites=sites,
            status=status,
            templates=templates,
            update_available=update_available,
            connected_by=connected_by,
        )

        return integration_word_press_page

