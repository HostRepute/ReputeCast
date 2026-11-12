from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.integration_site_woo_store_status import IntegrationSiteWooStoreStatus
from ..types import UNSET, Unset






T = TypeVar("T", bound="IntegrationSiteWoo")



@_attrs_define
class IntegrationSiteWoo:
    """ 
        Attributes:
            checkouts (bool):
            consent (bool):
            version (str):
            store_id (str | Unset):
            store_name (str | Unset):
            store_status (IntegrationSiteWooStoreStatus | Unset):
     """

    checkouts: bool
    consent: bool
    version: str
    store_id: str | Unset = UNSET
    store_name: str | Unset = UNSET
    store_status: IntegrationSiteWooStoreStatus | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        checkouts = self.checkouts

        consent = self.consent

        version = self.version

        store_id = self.store_id

        store_name = self.store_name

        store_status: str | Unset = UNSET
        if not isinstance(self.store_status, Unset):
            store_status = self.store_status.value



        field_dict: dict[str, Any] = {}

        field_dict.update({
            "checkouts": checkouts,
            "consent": consent,
            "version": version,
        })
        if store_id is not UNSET:
            field_dict["storeId"] = store_id
        if store_name is not UNSET:
            field_dict["storeName"] = store_name
        if store_status is not UNSET:
            field_dict["storeStatus"] = store_status

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        checkouts = d.pop("checkouts")

        consent = d.pop("consent")

        version = d.pop("version")

        store_id = d.pop("storeId", UNSET)

        store_name = d.pop("storeName", UNSET)

        _store_status = d.pop("storeStatus", UNSET)
        store_status: IntegrationSiteWooStoreStatus | Unset
        if isinstance(_store_status,  Unset):
            store_status = UNSET
        else:
            store_status = IntegrationSiteWooStoreStatus(_store_status)




        integration_site_woo = cls(
            checkouts=checkouts,
            consent=consent,
            version=version,
            store_id=store_id,
            store_name=store_name,
            store_status=store_status,
        )

        return integration_site_woo

