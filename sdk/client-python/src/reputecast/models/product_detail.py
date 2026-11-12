from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.product_detail_status import ProductDetailStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.product_check import ProductCheck
  from ..models.product_fields import ProductFields
  from ..models.product_file import ProductFile
  from ..models.product_list_option import ProductListOption
  from ..models.product_numbers import ProductNumbers
  from ..models.product_page_option import ProductPageOption
  from ..models.product_sales_page import ProductSalesPage
  from ..models.product_template_option import ProductTemplateOption
  from ..models.products_helpers import ProductsHelpers
  from ..models.products_payments import ProductsPayments





T = TypeVar("T", bound="ProductDetail")



@_attrs_define
class ProductDetail:
    """ 
        Attributes:
            changes (list[str]): The draft's fields that differ from what's live
            checks (list[ProductCheck]):
            chunk_bytes (int):
            created_by (str):
            draft (ProductFields):
            files (list[ProductFile]):
            files_bytes (int):
            helpers (ProductsHelpers):
            id (str):
            lists (list[ProductListOption]):
            max_bytes (int):
            max_file (int):
            orders (int):
            page_url (str):
            pages (list[ProductPageOption]):
            payments (ProductsPayments):
            published_by (str):
            route (str):
            sales_page (ProductSalesPage):
            shop_url (str):
            site_host (str):
            status (ProductDetailStatus):
            templates (list[ProductTemplateOption]):
            updated_at (datetime.datetime):
            updated_by (str):
            archived_at (datetime.datetime | Unset):
            live (ProductFields | Unset):
            numbers (ProductNumbers | Unset):
            published_at (datetime.datetime | Unset):
     """

    changes: list[str]
    checks: list[ProductCheck]
    chunk_bytes: int
    created_by: str
    draft: ProductFields
    files: list[ProductFile]
    files_bytes: int
    helpers: ProductsHelpers
    id: str
    lists: list[ProductListOption]
    max_bytes: int
    max_file: int
    orders: int
    page_url: str
    pages: list[ProductPageOption]
    payments: ProductsPayments
    published_by: str
    route: str
    sales_page: ProductSalesPage
    shop_url: str
    site_host: str
    status: ProductDetailStatus
    templates: list[ProductTemplateOption]
    updated_at: datetime.datetime
    updated_by: str
    archived_at: datetime.datetime | Unset = UNSET
    live: ProductFields | Unset = UNSET
    numbers: ProductNumbers | Unset = UNSET
    published_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.product_check import ProductCheck # noqa: PLC0415
        from ..models.product_fields import ProductFields # noqa: PLC0415
        from ..models.product_file import ProductFile # noqa: PLC0415
        from ..models.product_list_option import ProductListOption # noqa: PLC0415
        from ..models.product_numbers import ProductNumbers # noqa: PLC0415
        from ..models.product_page_option import ProductPageOption # noqa: PLC0415
        from ..models.product_sales_page import ProductSalesPage # noqa: PLC0415
        from ..models.product_template_option import ProductTemplateOption # noqa: PLC0415
        from ..models.products_helpers import ProductsHelpers # noqa: PLC0415
        from ..models.products_payments import ProductsPayments # noqa: PLC0415
        changes = self.changes



        checks = []
        for checks_item_data in self.checks:
            checks_item = checks_item_data.to_dict()
            checks.append(checks_item)



        chunk_bytes = self.chunk_bytes

        created_by = self.created_by

        draft = self.draft.to_dict()

        files = []
        for files_item_data in self.files:
            files_item = files_item_data.to_dict()
            files.append(files_item)



        files_bytes = self.files_bytes

        helpers = self.helpers.to_dict()

        id = self.id

        lists = []
        for lists_item_data in self.lists:
            lists_item = lists_item_data.to_dict()
            lists.append(lists_item)



        max_bytes = self.max_bytes

        max_file = self.max_file

        orders = self.orders

        page_url = self.page_url

        pages = []
        for pages_item_data in self.pages:
            pages_item = pages_item_data.to_dict()
            pages.append(pages_item)



        payments = self.payments.to_dict()

        published_by = self.published_by

        route = self.route

        sales_page = self.sales_page.to_dict()

        shop_url = self.shop_url

        site_host = self.site_host

        status = self.status.value

        templates = []
        for templates_item_data in self.templates:
            templates_item = templates_item_data.to_dict()
            templates.append(templates_item)



        updated_at = self.updated_at.isoformat()

        updated_by = self.updated_by

        archived_at: str | Unset = UNSET
        if not isinstance(self.archived_at, Unset):
            archived_at = self.archived_at.isoformat()

        live: dict[str, Any] | Unset = UNSET
        if not isinstance(self.live, Unset):
            live = self.live.to_dict()

        numbers: dict[str, Any] | Unset = UNSET
        if not isinstance(self.numbers, Unset):
            numbers = self.numbers.to_dict()

        published_at: str | Unset = UNSET
        if not isinstance(self.published_at, Unset):
            published_at = self.published_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "changes": changes,
            "checks": checks,
            "chunkBytes": chunk_bytes,
            "createdBy": created_by,
            "draft": draft,
            "files": files,
            "filesBytes": files_bytes,
            "helpers": helpers,
            "id": id,
            "lists": lists,
            "maxBytes": max_bytes,
            "maxFile": max_file,
            "orders": orders,
            "pageUrl": page_url,
            "pages": pages,
            "payments": payments,
            "publishedBy": published_by,
            "route": route,
            "salesPage": sales_page,
            "shopUrl": shop_url,
            "siteHost": site_host,
            "status": status,
            "templates": templates,
            "updatedAt": updated_at,
            "updatedBy": updated_by,
        })
        if archived_at is not UNSET:
            field_dict["archivedAt"] = archived_at
        if live is not UNSET:
            field_dict["live"] = live
        if numbers is not UNSET:
            field_dict["numbers"] = numbers
        if published_at is not UNSET:
            field_dict["publishedAt"] = published_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.product_check import ProductCheck # noqa: PLC0415
        from ..models.product_fields import ProductFields # noqa: PLC0415
        from ..models.product_file import ProductFile # noqa: PLC0415
        from ..models.product_list_option import ProductListOption # noqa: PLC0415
        from ..models.product_numbers import ProductNumbers # noqa: PLC0415
        from ..models.product_page_option import ProductPageOption # noqa: PLC0415
        from ..models.product_sales_page import ProductSalesPage # noqa: PLC0415
        from ..models.product_template_option import ProductTemplateOption # noqa: PLC0415
        from ..models.products_helpers import ProductsHelpers # noqa: PLC0415
        from ..models.products_payments import ProductsPayments # noqa: PLC0415
        d = dict(src_dict)
        changes = cast(list[str], d.pop("changes"))


        checks = []
        _checks = d.pop("checks")
        for checks_item_data in (_checks):
            checks_item = ProductCheck.from_dict(checks_item_data)



            checks.append(checks_item)


        chunk_bytes = d.pop("chunkBytes")

        created_by = d.pop("createdBy")

        draft = ProductFields.from_dict(d.pop("draft"))




        files = []
        _files = d.pop("files")
        for files_item_data in (_files):
            files_item = ProductFile.from_dict(files_item_data)



            files.append(files_item)


        files_bytes = d.pop("filesBytes")

        helpers = ProductsHelpers.from_dict(d.pop("helpers"))




        id = d.pop("id")

        lists = []
        _lists = d.pop("lists")
        for lists_item_data in (_lists):
            lists_item = ProductListOption.from_dict(lists_item_data)



            lists.append(lists_item)


        max_bytes = d.pop("maxBytes")

        max_file = d.pop("maxFile")

        orders = d.pop("orders")

        page_url = d.pop("pageUrl")

        pages = []
        _pages = d.pop("pages")
        for pages_item_data in (_pages):
            pages_item = ProductPageOption.from_dict(pages_item_data)



            pages.append(pages_item)


        payments = ProductsPayments.from_dict(d.pop("payments"))




        published_by = d.pop("publishedBy")

        route = d.pop("route")

        sales_page = ProductSalesPage.from_dict(d.pop("salesPage"))




        shop_url = d.pop("shopUrl")

        site_host = d.pop("siteHost")

        status = ProductDetailStatus(d.pop("status"))




        templates = []
        _templates = d.pop("templates")
        for templates_item_data in (_templates):
            templates_item = ProductTemplateOption.from_dict(templates_item_data)



            templates.append(templates_item)


        updated_at = datetime.datetime.fromisoformat(d.pop("updatedAt"))




        updated_by = d.pop("updatedBy")

        _archived_at = d.pop("archivedAt", UNSET)
        archived_at: datetime.datetime | Unset
        if isinstance(_archived_at,  Unset):
            archived_at = UNSET
        else:
            archived_at = datetime.datetime.fromisoformat(_archived_at)




        _live = d.pop("live", UNSET)
        live: ProductFields | Unset
        if isinstance(_live,  Unset):
            live = UNSET
        else:
            live = ProductFields.from_dict(_live)




        _numbers = d.pop("numbers", UNSET)
        numbers: ProductNumbers | Unset
        if isinstance(_numbers,  Unset):
            numbers = UNSET
        else:
            numbers = ProductNumbers.from_dict(_numbers)




        _published_at = d.pop("publishedAt", UNSET)
        published_at: datetime.datetime | Unset
        if isinstance(_published_at,  Unset):
            published_at = UNSET
        else:
            published_at = datetime.datetime.fromisoformat(_published_at)




        product_detail = cls(
            changes=changes,
            checks=checks,
            chunk_bytes=chunk_bytes,
            created_by=created_by,
            draft=draft,
            files=files,
            files_bytes=files_bytes,
            helpers=helpers,
            id=id,
            lists=lists,
            max_bytes=max_bytes,
            max_file=max_file,
            orders=orders,
            page_url=page_url,
            pages=pages,
            payments=payments,
            published_by=published_by,
            route=route,
            sales_page=sales_page,
            shop_url=shop_url,
            site_host=site_host,
            status=status,
            templates=templates,
            updated_at=updated_at,
            updated_by=updated_by,
            archived_at=archived_at,
            live=live,
            numbers=numbers,
            published_at=published_at,
        )

        return product_detail

