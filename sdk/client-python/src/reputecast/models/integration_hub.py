from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.integration_alert import IntegrationAlert
  from ..models.integration_group import IntegrationGroup





T = TypeVar("T", bound="IntegrationHub")



@_attrs_define
class IntegrationHub:
    """ 
        Attributes:
            alerts (list[IntegrationAlert]):
            can_manage (bool):
            groups (list[IntegrationGroup]): Groups with something in them, in order
            helpers (list[str]):
     """

    alerts: list[IntegrationAlert]
    can_manage: bool
    groups: list[IntegrationGroup]
    helpers: list[str]





    def to_dict(self) -> dict[str, Any]:
        from ..models.integration_alert import IntegrationAlert # noqa: PLC0415
        from ..models.integration_group import IntegrationGroup # noqa: PLC0415
        alerts = []
        for alerts_item_data in self.alerts:
            alerts_item = alerts_item_data.to_dict()
            alerts.append(alerts_item)



        can_manage = self.can_manage

        groups = []
        for groups_item_data in self.groups:
            groups_item = groups_item_data.to_dict()
            groups.append(groups_item)



        helpers = self.helpers




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "alerts": alerts,
            "canManage": can_manage,
            "groups": groups,
            "helpers": helpers,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.integration_alert import IntegrationAlert # noqa: PLC0415
        from ..models.integration_group import IntegrationGroup # noqa: PLC0415
        d = dict(src_dict)
        alerts = []
        _alerts = d.pop("alerts")
        for alerts_item_data in (_alerts):
            alerts_item = IntegrationAlert.from_dict(alerts_item_data)



            alerts.append(alerts_item)


        can_manage = d.pop("canManage")

        groups = []
        _groups = d.pop("groups")
        for groups_item_data in (_groups):
            groups_item = IntegrationGroup.from_dict(groups_item_data)



            groups.append(groups_item)


        helpers = cast(list[str], d.pop("helpers"))


        integration_hub = cls(
            alerts=alerts,
            can_manage=can_manage,
            groups=groups,
            helpers=helpers,
        )

        return integration_hub

