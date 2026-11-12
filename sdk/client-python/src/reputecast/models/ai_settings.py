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
  from ..models.ai_connection_view import AIConnectionView
  from ..models.ai_feature_state import AIFeatureState
  from ..models.ai_member_role import AIMemberRole
  from ..models.ai_provider_entry import AIProviderEntry
  from ..models.ai_usage_feature import AIUsageFeature
  from ..models.ai_usage_member import AIUsageMember
  from ..models.ai_usage_view import AIUsageView





T = TypeVar("T", bound="AISettings")



@_attrs_define
class AISettings:
    """ 
        Attributes:
            ai_use_roles (list[str]):
            allow_private (bool):
            by_feature (list[AIUsageFeature]):
            by_member (list[AIUsageMember]):
            can_manage (bool):
            catalog (list[AIProviderEntry]):
            connected (bool):
            disconnected (bool):
            features (list[AIFeatureState]):
            helpers (list[str]):
            instance_admins (list[str]):
            price_list_date (datetime.datetime):
            usage (AIUsageView):
            without_ai_use (list[AIMemberRole]):
            connection (AIConnectionView | Unset):
     """

    ai_use_roles: list[str]
    allow_private: bool
    by_feature: list[AIUsageFeature]
    by_member: list[AIUsageMember]
    can_manage: bool
    catalog: list[AIProviderEntry]
    connected: bool
    disconnected: bool
    features: list[AIFeatureState]
    helpers: list[str]
    instance_admins: list[str]
    price_list_date: datetime.datetime
    usage: AIUsageView
    without_ai_use: list[AIMemberRole]
    connection: AIConnectionView | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.ai_connection_view import AIConnectionView # noqa: PLC0415
        from ..models.ai_feature_state import AIFeatureState # noqa: PLC0415
        from ..models.ai_member_role import AIMemberRole # noqa: PLC0415
        from ..models.ai_provider_entry import AIProviderEntry # noqa: PLC0415
        from ..models.ai_usage_feature import AIUsageFeature # noqa: PLC0415
        from ..models.ai_usage_member import AIUsageMember # noqa: PLC0415
        from ..models.ai_usage_view import AIUsageView # noqa: PLC0415
        ai_use_roles = self.ai_use_roles



        allow_private = self.allow_private

        by_feature = []
        for by_feature_item_data in self.by_feature:
            by_feature_item = by_feature_item_data.to_dict()
            by_feature.append(by_feature_item)



        by_member = []
        for by_member_item_data in self.by_member:
            by_member_item = by_member_item_data.to_dict()
            by_member.append(by_member_item)



        can_manage = self.can_manage

        catalog = []
        for catalog_item_data in self.catalog:
            catalog_item = catalog_item_data.to_dict()
            catalog.append(catalog_item)



        connected = self.connected

        disconnected = self.disconnected

        features = []
        for features_item_data in self.features:
            features_item = features_item_data.to_dict()
            features.append(features_item)



        helpers = self.helpers



        instance_admins = self.instance_admins



        price_list_date = self.price_list_date.isoformat()

        usage = self.usage.to_dict()

        without_ai_use = []
        for without_ai_use_item_data in self.without_ai_use:
            without_ai_use_item = without_ai_use_item_data.to_dict()
            without_ai_use.append(without_ai_use_item)



        connection: dict[str, Any] | Unset = UNSET
        if not isinstance(self.connection, Unset):
            connection = self.connection.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "aiUseRoles": ai_use_roles,
            "allowPrivate": allow_private,
            "byFeature": by_feature,
            "byMember": by_member,
            "canManage": can_manage,
            "catalog": catalog,
            "connected": connected,
            "disconnected": disconnected,
            "features": features,
            "helpers": helpers,
            "instanceAdmins": instance_admins,
            "priceListDate": price_list_date,
            "usage": usage,
            "withoutAiUse": without_ai_use,
        })
        if connection is not UNSET:
            field_dict["connection"] = connection

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.ai_connection_view import AIConnectionView # noqa: PLC0415
        from ..models.ai_feature_state import AIFeatureState # noqa: PLC0415
        from ..models.ai_member_role import AIMemberRole # noqa: PLC0415
        from ..models.ai_provider_entry import AIProviderEntry # noqa: PLC0415
        from ..models.ai_usage_feature import AIUsageFeature # noqa: PLC0415
        from ..models.ai_usage_member import AIUsageMember # noqa: PLC0415
        from ..models.ai_usage_view import AIUsageView # noqa: PLC0415
        d = dict(src_dict)
        ai_use_roles = cast(list[str], d.pop("aiUseRoles"))


        allow_private = d.pop("allowPrivate")

        by_feature = []
        _by_feature = d.pop("byFeature")
        for by_feature_item_data in (_by_feature):
            by_feature_item = AIUsageFeature.from_dict(by_feature_item_data)



            by_feature.append(by_feature_item)


        by_member = []
        _by_member = d.pop("byMember")
        for by_member_item_data in (_by_member):
            by_member_item = AIUsageMember.from_dict(by_member_item_data)



            by_member.append(by_member_item)


        can_manage = d.pop("canManage")

        catalog = []
        _catalog = d.pop("catalog")
        for catalog_item_data in (_catalog):
            catalog_item = AIProviderEntry.from_dict(catalog_item_data)



            catalog.append(catalog_item)


        connected = d.pop("connected")

        disconnected = d.pop("disconnected")

        features = []
        _features = d.pop("features")
        for features_item_data in (_features):
            features_item = AIFeatureState.from_dict(features_item_data)



            features.append(features_item)


        helpers = cast(list[str], d.pop("helpers"))


        instance_admins = cast(list[str], d.pop("instanceAdmins"))


        price_list_date = datetime.datetime.fromisoformat(d.pop("priceListDate"))




        usage = AIUsageView.from_dict(d.pop("usage"))




        without_ai_use = []
        _without_ai_use = d.pop("withoutAiUse")
        for without_ai_use_item_data in (_without_ai_use):
            without_ai_use_item = AIMemberRole.from_dict(without_ai_use_item_data)



            without_ai_use.append(without_ai_use_item)


        _connection = d.pop("connection", UNSET)
        connection: AIConnectionView | Unset
        if isinstance(_connection,  Unset):
            connection = UNSET
        else:
            connection = AIConnectionView.from_dict(_connection)




        ai_settings = cls(
            ai_use_roles=ai_use_roles,
            allow_private=allow_private,
            by_feature=by_feature,
            by_member=by_member,
            can_manage=can_manage,
            catalog=catalog,
            connected=connected,
            disconnected=disconnected,
            features=features,
            helpers=helpers,
            instance_admins=instance_admins,
            price_list_date=price_list_date,
            usage=usage,
            without_ai_use=without_ai_use,
            connection=connection,
        )

        return ai_settings

