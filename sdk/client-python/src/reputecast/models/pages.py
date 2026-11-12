from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.archive_page import ArchivePage
  from ..models.confirm_page import ConfirmPage
  from ..models.preferences_page import PreferencesPage
  from ..models.subscribe_page import SubscribePage
  from ..models.unsubscribe_page import UnsubscribePage





T = TypeVar("T", bound="Pages")



@_attrs_define
class Pages:
    """ 
        Attributes:
            archive (ArchivePage):
            confirm (ConfirmPage):
            preferences (PreferencesPage):
            preview_email (str):
            subscribe (SubscribePage):
            unsubscribe (UnsubscribePage):
     """

    archive: ArchivePage
    confirm: ConfirmPage
    preferences: PreferencesPage
    preview_email: str
    subscribe: SubscribePage
    unsubscribe: UnsubscribePage





    def to_dict(self) -> dict[str, Any]:
        from ..models.archive_page import ArchivePage # noqa: PLC0415
        from ..models.confirm_page import ConfirmPage # noqa: PLC0415
        from ..models.preferences_page import PreferencesPage # noqa: PLC0415
        from ..models.subscribe_page import SubscribePage # noqa: PLC0415
        from ..models.unsubscribe_page import UnsubscribePage # noqa: PLC0415
        archive = self.archive.to_dict()

        confirm = self.confirm.to_dict()

        preferences = self.preferences.to_dict()

        preview_email = self.preview_email

        subscribe = self.subscribe.to_dict()

        unsubscribe = self.unsubscribe.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "archive": archive,
            "confirm": confirm,
            "preferences": preferences,
            "previewEmail": preview_email,
            "subscribe": subscribe,
            "unsubscribe": unsubscribe,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.archive_page import ArchivePage # noqa: PLC0415
        from ..models.confirm_page import ConfirmPage # noqa: PLC0415
        from ..models.preferences_page import PreferencesPage # noqa: PLC0415
        from ..models.subscribe_page import SubscribePage # noqa: PLC0415
        from ..models.unsubscribe_page import UnsubscribePage # noqa: PLC0415
        d = dict(src_dict)
        archive = ArchivePage.from_dict(d.pop("archive"))




        confirm = ConfirmPage.from_dict(d.pop("confirm"))




        preferences = PreferencesPage.from_dict(d.pop("preferences"))




        preview_email = d.pop("previewEmail")

        subscribe = SubscribePage.from_dict(d.pop("subscribe"))




        unsubscribe = UnsubscribePage.from_dict(d.pop("unsubscribe"))




        pages = cls(
            archive=archive,
            confirm=confirm,
            preferences=preferences,
            preview_email=preview_email,
            subscribe=subscribe,
            unsubscribe=unsubscribe,
        )

        return pages

