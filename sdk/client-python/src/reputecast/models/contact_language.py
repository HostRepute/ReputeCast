from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.contact_language_source import ContactLanguageSource
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.contact_language_option import ContactLanguageOption





T = TypeVar("T", bound="ContactLanguage")



@_attrs_define
class ContactLanguage:
    """ 
        Attributes:
            browser (bool): A signup form read it from the browser
            code (str):
            gets (str):
            name (str):
            offered (bool):
            options (list[ContactLanguageOption]):
            source (ContactLanguageSource):
            source_name (str):
            set_at (datetime.datetime | Unset):
     """

    browser: bool
    code: str
    gets: str
    name: str
    offered: bool
    options: list[ContactLanguageOption]
    source: ContactLanguageSource
    source_name: str
    set_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.contact_language_option import ContactLanguageOption # noqa: PLC0415
        browser = self.browser

        code = self.code

        gets = self.gets

        name = self.name

        offered = self.offered

        options = []
        for options_item_data in self.options:
            options_item = options_item_data.to_dict()
            options.append(options_item)



        source = self.source.value

        source_name = self.source_name

        set_at: str | Unset = UNSET
        if not isinstance(self.set_at, Unset):
            set_at = self.set_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "browser": browser,
            "code": code,
            "gets": gets,
            "name": name,
            "offered": offered,
            "options": options,
            "source": source,
            "sourceName": source_name,
        })
        if set_at is not UNSET:
            field_dict["setAt"] = set_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.contact_language_option import ContactLanguageOption # noqa: PLC0415
        d = dict(src_dict)
        browser = d.pop("browser")

        code = d.pop("code")

        gets = d.pop("gets")

        name = d.pop("name")

        offered = d.pop("offered")

        options = []
        _options = d.pop("options")
        for options_item_data in (_options):
            options_item = ContactLanguageOption.from_dict(options_item_data)



            options.append(options_item)


        source = ContactLanguageSource(d.pop("source"))




        source_name = d.pop("sourceName")

        _set_at = d.pop("setAt", UNSET)
        set_at: datetime.datetime | Unset
        if isinstance(_set_at,  Unset):
            set_at = UNSET
        else:
            set_at = datetime.datetime.fromisoformat(_set_at)




        contact_language = cls(
            browser=browser,
            code=code,
            gets=gets,
            name=name,
            offered=offered,
            options=options,
            source=source,
            source_name=source_name,
            set_at=set_at,
        )

        return contact_language

