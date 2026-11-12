from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.gifts_view_mode import GiftsViewMode
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.gift_email import GiftEmail
  from ..models.gift_ending_warning import GiftEndingWarning
  from ..models.gift_numbers import GiftNumbers
  from ..models.gift_option import GiftOption
  from ..models.gift_page import GiftPage
  from ..models.gift_row import GiftRow
  from ..models.gift_tier import GiftTier





T = TypeVar("T", bound="GiftsView")



@_attrs_define
class GiftsView:
    """ 
        Attributes:
            can_edit (bool):
            can_manage (bool):
            category_kind (str):
            category_name (str):
            connected (bool):
            currency (str):
            emails (list[GiftEmail]):
            from_email (str):
            from_name (str):
            helpers (list[str]):
            list_names (list[str]):
            mode (GiftsViewMode):
            needs_look (list[GiftRow]):
            numbers (GiftNumbers):
            options (list[GiftOption]):
            page (GiftPage):
            provider (str):
            route_primary (str):
            tiers (list[GiftTier]):
            ending (GiftEndingWarning | Unset):
     """

    can_edit: bool
    can_manage: bool
    category_kind: str
    category_name: str
    connected: bool
    currency: str
    emails: list[GiftEmail]
    from_email: str
    from_name: str
    helpers: list[str]
    list_names: list[str]
    mode: GiftsViewMode
    needs_look: list[GiftRow]
    numbers: GiftNumbers
    options: list[GiftOption]
    page: GiftPage
    provider: str
    route_primary: str
    tiers: list[GiftTier]
    ending: GiftEndingWarning | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.gift_email import GiftEmail # noqa: PLC0415
        from ..models.gift_ending_warning import GiftEndingWarning # noqa: PLC0415
        from ..models.gift_numbers import GiftNumbers # noqa: PLC0415
        from ..models.gift_option import GiftOption # noqa: PLC0415
        from ..models.gift_page import GiftPage # noqa: PLC0415
        from ..models.gift_row import GiftRow # noqa: PLC0415
        from ..models.gift_tier import GiftTier # noqa: PLC0415
        can_edit = self.can_edit

        can_manage = self.can_manage

        category_kind = self.category_kind

        category_name = self.category_name

        connected = self.connected

        currency = self.currency

        emails = []
        for emails_item_data in self.emails:
            emails_item = emails_item_data.to_dict()
            emails.append(emails_item)



        from_email = self.from_email

        from_name = self.from_name

        helpers = self.helpers



        list_names = self.list_names



        mode = self.mode.value

        needs_look = []
        for needs_look_item_data in self.needs_look:
            needs_look_item = needs_look_item_data.to_dict()
            needs_look.append(needs_look_item)



        numbers = self.numbers.to_dict()

        options = []
        for options_item_data in self.options:
            options_item = options_item_data.to_dict()
            options.append(options_item)



        page = self.page.to_dict()

        provider = self.provider

        route_primary = self.route_primary

        tiers = []
        for tiers_item_data in self.tiers:
            tiers_item = tiers_item_data.to_dict()
            tiers.append(tiers_item)



        ending: dict[str, Any] | Unset = UNSET
        if not isinstance(self.ending, Unset):
            ending = self.ending.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canEdit": can_edit,
            "canManage": can_manage,
            "categoryKind": category_kind,
            "categoryName": category_name,
            "connected": connected,
            "currency": currency,
            "emails": emails,
            "fromEmail": from_email,
            "fromName": from_name,
            "helpers": helpers,
            "listNames": list_names,
            "mode": mode,
            "needsLook": needs_look,
            "numbers": numbers,
            "options": options,
            "page": page,
            "provider": provider,
            "routePrimary": route_primary,
            "tiers": tiers,
        })
        if ending is not UNSET:
            field_dict["ending"] = ending

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.gift_email import GiftEmail # noqa: PLC0415
        from ..models.gift_ending_warning import GiftEndingWarning # noqa: PLC0415
        from ..models.gift_numbers import GiftNumbers # noqa: PLC0415
        from ..models.gift_option import GiftOption # noqa: PLC0415
        from ..models.gift_page import GiftPage # noqa: PLC0415
        from ..models.gift_row import GiftRow # noqa: PLC0415
        from ..models.gift_tier import GiftTier # noqa: PLC0415
        d = dict(src_dict)
        can_edit = d.pop("canEdit")

        can_manage = d.pop("canManage")

        category_kind = d.pop("categoryKind")

        category_name = d.pop("categoryName")

        connected = d.pop("connected")

        currency = d.pop("currency")

        emails = []
        _emails = d.pop("emails")
        for emails_item_data in (_emails):
            emails_item = GiftEmail.from_dict(emails_item_data)



            emails.append(emails_item)


        from_email = d.pop("fromEmail")

        from_name = d.pop("fromName")

        helpers = cast(list[str], d.pop("helpers"))


        list_names = cast(list[str], d.pop("listNames"))


        mode = GiftsViewMode(d.pop("mode"))




        needs_look = []
        _needs_look = d.pop("needsLook")
        for needs_look_item_data in (_needs_look):
            needs_look_item = GiftRow.from_dict(needs_look_item_data)



            needs_look.append(needs_look_item)


        numbers = GiftNumbers.from_dict(d.pop("numbers"))




        options = []
        _options = d.pop("options")
        for options_item_data in (_options):
            options_item = GiftOption.from_dict(options_item_data)



            options.append(options_item)


        page = GiftPage.from_dict(d.pop("page"))




        provider = d.pop("provider")

        route_primary = d.pop("routePrimary")

        tiers = []
        _tiers = d.pop("tiers")
        for tiers_item_data in (_tiers):
            tiers_item = GiftTier.from_dict(tiers_item_data)



            tiers.append(tiers_item)


        _ending = d.pop("ending", UNSET)
        ending: GiftEndingWarning | Unset
        if isinstance(_ending,  Unset):
            ending = UNSET
        else:
            ending = GiftEndingWarning.from_dict(_ending)




        gifts_view = cls(
            can_edit=can_edit,
            can_manage=can_manage,
            category_kind=category_kind,
            category_name=category_name,
            connected=connected,
            currency=currency,
            emails=emails,
            from_email=from_email,
            from_name=from_name,
            helpers=helpers,
            list_names=list_names,
            mode=mode,
            needs_look=needs_look,
            numbers=numbers,
            options=options,
            page=page,
            provider=provider,
            route_primary=route_primary,
            tiers=tiers,
            ending=ending,
        )

        return gifts_view

