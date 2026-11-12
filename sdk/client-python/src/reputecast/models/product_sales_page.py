from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.product_sales_page_status import ProductSalesPageStatus
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ProductSalesPage")



@_attrs_define
class ProductSalesPage:
    """ 
        Attributes:
            name (str):
            path (str):  Example: /field-guide.
            status (ProductSalesPageStatus):
            page_id (str | Unset): Empty for the default product page
            published_at (datetime.datetime | Unset):
     """

    name: str
    path: str
    status: ProductSalesPageStatus
    page_id: str | Unset = UNSET
    published_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        name = self.name

        path = self.path

        status = self.status.value

        page_id = self.page_id

        published_at: str | Unset = UNSET
        if not isinstance(self.published_at, Unset):
            published_at = self.published_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "name": name,
            "path": path,
            "status": status,
        })
        if page_id is not UNSET:
            field_dict["pageId"] = page_id
        if published_at is not UNSET:
            field_dict["publishedAt"] = published_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        name = d.pop("name")

        path = d.pop("path")

        status = ProductSalesPageStatus(d.pop("status"))




        page_id = d.pop("pageId", UNSET)

        _published_at = d.pop("publishedAt", UNSET)
        published_at: datetime.datetime | Unset
        if isinstance(_published_at,  Unset):
            published_at = UNSET
        else:
            published_at = datetime.datetime.fromisoformat(_published_at)




        product_sales_page = cls(
            name=name,
            path=path,
            status=status,
            page_id=page_id,
            published_at=published_at,
        )

        return product_sales_page

