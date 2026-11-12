from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.products_overview_period import ProductsOverviewPeriod
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.product_list_item import ProductListItem
  from ..models.product_source_row import ProductSourceRow
  from ..models.products_buyers import ProductsBuyers
  from ..models.products_helpers import ProductsHelpers
  from ..models.products_payments import ProductsPayments
  from ..models.products_stats import ProductsStats





T = TypeVar("T", bound="ProductsOverview")



@_attrs_define
class ProductsOverview:
    """ 
        Attributes:
            buyers (ProductsBuyers):
            from_ (datetime.datetime):
            helpers (ProductsHelpers):
            payments (ProductsPayments):
            period (ProductsOverviewPeriod):
            products (list[ProductListItem]):
            sources (list[ProductSourceRow]):
            stats (ProductsStats):
            until (datetime.datetime):
     """

    buyers: ProductsBuyers
    from_: datetime.datetime
    helpers: ProductsHelpers
    payments: ProductsPayments
    period: ProductsOverviewPeriod
    products: list[ProductListItem]
    sources: list[ProductSourceRow]
    stats: ProductsStats
    until: datetime.datetime





    def to_dict(self) -> dict[str, Any]:
        from ..models.product_list_item import ProductListItem # noqa: PLC0415
        from ..models.product_source_row import ProductSourceRow # noqa: PLC0415
        from ..models.products_buyers import ProductsBuyers # noqa: PLC0415
        from ..models.products_helpers import ProductsHelpers # noqa: PLC0415
        from ..models.products_payments import ProductsPayments # noqa: PLC0415
        from ..models.products_stats import ProductsStats # noqa: PLC0415
        buyers = self.buyers.to_dict()

        from_ = self.from_.isoformat()

        helpers = self.helpers.to_dict()

        payments = self.payments.to_dict()

        period = self.period.value

        products = []
        for products_item_data in self.products:
            products_item = products_item_data.to_dict()
            products.append(products_item)



        sources = []
        for sources_item_data in self.sources:
            sources_item = sources_item_data.to_dict()
            sources.append(sources_item)



        stats = self.stats.to_dict()

        until = self.until.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "buyers": buyers,
            "from": from_,
            "helpers": helpers,
            "payments": payments,
            "period": period,
            "products": products,
            "sources": sources,
            "stats": stats,
            "until": until,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.product_list_item import ProductListItem # noqa: PLC0415
        from ..models.product_source_row import ProductSourceRow # noqa: PLC0415
        from ..models.products_buyers import ProductsBuyers # noqa: PLC0415
        from ..models.products_helpers import ProductsHelpers # noqa: PLC0415
        from ..models.products_payments import ProductsPayments # noqa: PLC0415
        from ..models.products_stats import ProductsStats # noqa: PLC0415
        d = dict(src_dict)
        buyers = ProductsBuyers.from_dict(d.pop("buyers"))




        from_ = datetime.datetime.fromisoformat(d.pop("from"))




        helpers = ProductsHelpers.from_dict(d.pop("helpers"))




        payments = ProductsPayments.from_dict(d.pop("payments"))




        period = ProductsOverviewPeriod(d.pop("period"))




        products = []
        _products = d.pop("products")
        for products_item_data in (_products):
            products_item = ProductListItem.from_dict(products_item_data)



            products.append(products_item)


        sources = []
        _sources = d.pop("sources")
        for sources_item_data in (_sources):
            sources_item = ProductSourceRow.from_dict(sources_item_data)



            sources.append(sources_item)


        stats = ProductsStats.from_dict(d.pop("stats"))




        until = datetime.datetime.fromisoformat(d.pop("until"))




        products_overview = cls(
            buyers=buyers,
            from_=from_,
            helpers=helpers,
            payments=payments,
            period=period,
            products=products,
            sources=sources,
            stats=stats,
            until=until,
        )

        return products_overview

