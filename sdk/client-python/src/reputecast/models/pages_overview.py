from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.page_list_item import PageListItem
  from ..models.page_template_count import PageTemplateCount
  from ..models.pages_shop import PagesShop
  from ..models.pages_stats import PagesStats
  from ..models.pages_theme import PagesTheme





T = TypeVar("T", bound="PagesOverview")



@_attrs_define
class PagesOverview:
    """ 
        Attributes:
            can_edit (bool):
            can_publish (bool):
            editors (list[str]):
            from_ (datetime.datetime):
            host (str):
            pages (list[PageListItem]):
            period (str):
            publishers (list[str]):
            shop (PagesShop):
            stats (PagesStats):
            templates (list[PageTemplateCount]):
            theme (PagesTheme):
            until (datetime.datetime):
            website_on (bool):
     """

    can_edit: bool
    can_publish: bool
    editors: list[str]
    from_: datetime.datetime
    host: str
    pages: list[PageListItem]
    period: str
    publishers: list[str]
    shop: PagesShop
    stats: PagesStats
    templates: list[PageTemplateCount]
    theme: PagesTheme
    until: datetime.datetime
    website_on: bool





    def to_dict(self) -> dict[str, Any]:
        from ..models.page_list_item import PageListItem # noqa: PLC0415
        from ..models.page_template_count import PageTemplateCount # noqa: PLC0415
        from ..models.pages_shop import PagesShop # noqa: PLC0415
        from ..models.pages_stats import PagesStats # noqa: PLC0415
        from ..models.pages_theme import PagesTheme # noqa: PLC0415
        can_edit = self.can_edit

        can_publish = self.can_publish

        editors = self.editors



        from_ = self.from_.isoformat()

        host = self.host

        pages = []
        for pages_item_data in self.pages:
            pages_item = pages_item_data.to_dict()
            pages.append(pages_item)



        period = self.period

        publishers = self.publishers



        shop = self.shop.to_dict()

        stats = self.stats.to_dict()

        templates = []
        for templates_item_data in self.templates:
            templates_item = templates_item_data.to_dict()
            templates.append(templates_item)



        theme = self.theme.to_dict()

        until = self.until.isoformat()

        website_on = self.website_on


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canEdit": can_edit,
            "canPublish": can_publish,
            "editors": editors,
            "from": from_,
            "host": host,
            "pages": pages,
            "period": period,
            "publishers": publishers,
            "shop": shop,
            "stats": stats,
            "templates": templates,
            "theme": theme,
            "until": until,
            "websiteOn": website_on,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.page_list_item import PageListItem # noqa: PLC0415
        from ..models.page_template_count import PageTemplateCount # noqa: PLC0415
        from ..models.pages_shop import PagesShop # noqa: PLC0415
        from ..models.pages_stats import PagesStats # noqa: PLC0415
        from ..models.pages_theme import PagesTheme # noqa: PLC0415
        d = dict(src_dict)
        can_edit = d.pop("canEdit")

        can_publish = d.pop("canPublish")

        editors = cast(list[str], d.pop("editors"))


        from_ = datetime.datetime.fromisoformat(d.pop("from"))




        host = d.pop("host")

        pages = []
        _pages = d.pop("pages")
        for pages_item_data in (_pages):
            pages_item = PageListItem.from_dict(pages_item_data)



            pages.append(pages_item)


        period = d.pop("period")

        publishers = cast(list[str], d.pop("publishers"))


        shop = PagesShop.from_dict(d.pop("shop"))




        stats = PagesStats.from_dict(d.pop("stats"))




        templates = []
        _templates = d.pop("templates")
        for templates_item_data in (_templates):
            templates_item = PageTemplateCount.from_dict(templates_item_data)



            templates.append(templates_item)


        theme = PagesTheme.from_dict(d.pop("theme"))




        until = datetime.datetime.fromisoformat(d.pop("until"))




        website_on = d.pop("websiteOn")

        pages_overview = cls(
            can_edit=can_edit,
            can_publish=can_publish,
            editors=editors,
            from_=from_,
            host=host,
            pages=pages,
            period=period,
            publishers=publishers,
            shop=shop,
            stats=stats,
            templates=templates,
            theme=theme,
            until=until,
            website_on=website_on,
        )

        return pages_overview

