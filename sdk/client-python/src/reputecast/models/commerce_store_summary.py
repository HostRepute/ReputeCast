from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.commerce_store_summary_platform import CommerceStoreSummaryPlatform
from ..models.commerce_store_summary_status import CommerceStoreSummaryStatus
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.commerce_sync_state import CommerceSyncState





T = TypeVar("T", bound="CommerceStoreSummary")



@_attrs_define
class CommerceStoreSummary:
    """ 
        Attributes:
            address (str):
            currency (str):
            demo (bool):
            domain (str):
            id (str):
            name (str):
            platform (CommerceStoreSummaryPlatform):
            products (int):
            status (CommerceStoreSummaryStatus):
            step (str):
            sync (CommerceSyncState):
            list_id (str | Unset):
            list_name (str | Unset):
     """

    address: str
    currency: str
    demo: bool
    domain: str
    id: str
    name: str
    platform: CommerceStoreSummaryPlatform
    products: int
    status: CommerceStoreSummaryStatus
    step: str
    sync: CommerceSyncState
    list_id: str | Unset = UNSET
    list_name: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.commerce_sync_state import CommerceSyncState # noqa: PLC0415
        address = self.address

        currency = self.currency

        demo = self.demo

        domain = self.domain

        id = self.id

        name = self.name

        platform = self.platform.value

        products = self.products

        status = self.status.value

        step = self.step

        sync = self.sync.to_dict()

        list_id = self.list_id

        list_name = self.list_name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "address": address,
            "currency": currency,
            "demo": demo,
            "domain": domain,
            "id": id,
            "name": name,
            "platform": platform,
            "products": products,
            "status": status,
            "step": step,
            "sync": sync,
        })
        if list_id is not UNSET:
            field_dict["listId"] = list_id
        if list_name is not UNSET:
            field_dict["listName"] = list_name

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.commerce_sync_state import CommerceSyncState # noqa: PLC0415
        d = dict(src_dict)
        address = d.pop("address")

        currency = d.pop("currency")

        demo = d.pop("demo")

        domain = d.pop("domain")

        id = d.pop("id")

        name = d.pop("name")

        platform = CommerceStoreSummaryPlatform(d.pop("platform"))




        products = d.pop("products")

        status = CommerceStoreSummaryStatus(d.pop("status"))




        step = d.pop("step")

        sync = CommerceSyncState.from_dict(d.pop("sync"))




        list_id = d.pop("listId", UNSET)

        list_name = d.pop("listName", UNSET)

        commerce_store_summary = cls(
            address=address,
            currency=currency,
            demo=demo,
            domain=domain,
            id=id,
            name=name,
            platform=platform,
            products=products,
            status=status,
            step=step,
            sync=sync,
            list_id=list_id,
            list_name=list_name,
        )

        return commerce_store_summary

