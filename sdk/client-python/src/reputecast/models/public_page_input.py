from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.archive_page import ArchivePage
  from ..models.confirm_page import ConfirmPage
  from ..models.preferences_page import PreferencesPage
  from ..models.subscribe_page import SubscribePage
  from ..models.unsubscribe_page import UnsubscribePage





T = TypeVar("T", bound="PublicPageInput")



@_attrs_define
class PublicPageInput:
    """ 
        Attributes:
            archive (ArchivePage | Unset):
            confirm (ConfirmPage | Unset):
            hosted_form_id (str | Unset):
            preferences (PreferencesPage | Unset):
            preview_email (str | Unset):
            subscribe (SubscribePage | Unset):
            unsubscribe (UnsubscribePage | Unset):
     """

    archive: ArchivePage | Unset = UNSET
    confirm: ConfirmPage | Unset = UNSET
    hosted_form_id: str | Unset = UNSET
    preferences: PreferencesPage | Unset = UNSET
    preview_email: str | Unset = UNSET
    subscribe: SubscribePage | Unset = UNSET
    unsubscribe: UnsubscribePage | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.archive_page import ArchivePage # noqa: PLC0415
        from ..models.confirm_page import ConfirmPage # noqa: PLC0415
        from ..models.preferences_page import PreferencesPage # noqa: PLC0415
        from ..models.subscribe_page import SubscribePage # noqa: PLC0415
        from ..models.unsubscribe_page import UnsubscribePage # noqa: PLC0415
        archive: dict[str, Any] | Unset = UNSET
        if not isinstance(self.archive, Unset):
            archive = self.archive.to_dict()

        confirm: dict[str, Any] | Unset = UNSET
        if not isinstance(self.confirm, Unset):
            confirm = self.confirm.to_dict()

        hosted_form_id = self.hosted_form_id

        preferences: dict[str, Any] | Unset = UNSET
        if not isinstance(self.preferences, Unset):
            preferences = self.preferences.to_dict()

        preview_email = self.preview_email

        subscribe: dict[str, Any] | Unset = UNSET
        if not isinstance(self.subscribe, Unset):
            subscribe = self.subscribe.to_dict()

        unsubscribe: dict[str, Any] | Unset = UNSET
        if not isinstance(self.unsubscribe, Unset):
            unsubscribe = self.unsubscribe.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if archive is not UNSET:
            field_dict["archive"] = archive
        if confirm is not UNSET:
            field_dict["confirm"] = confirm
        if hosted_form_id is not UNSET:
            field_dict["hostedFormId"] = hosted_form_id
        if preferences is not UNSET:
            field_dict["preferences"] = preferences
        if preview_email is not UNSET:
            field_dict["previewEmail"] = preview_email
        if subscribe is not UNSET:
            field_dict["subscribe"] = subscribe
        if unsubscribe is not UNSET:
            field_dict["unsubscribe"] = unsubscribe

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.archive_page import ArchivePage # noqa: PLC0415
        from ..models.confirm_page import ConfirmPage # noqa: PLC0415
        from ..models.preferences_page import PreferencesPage # noqa: PLC0415
        from ..models.subscribe_page import SubscribePage # noqa: PLC0415
        from ..models.unsubscribe_page import UnsubscribePage # noqa: PLC0415
        d = dict(src_dict)
        _archive = d.pop("archive", UNSET)
        archive: ArchivePage | Unset
        if isinstance(_archive,  Unset):
            archive = UNSET
        else:
            archive = ArchivePage.from_dict(_archive)




        _confirm = d.pop("confirm", UNSET)
        confirm: ConfirmPage | Unset
        if isinstance(_confirm,  Unset):
            confirm = UNSET
        else:
            confirm = ConfirmPage.from_dict(_confirm)




        hosted_form_id = d.pop("hostedFormId", UNSET)

        _preferences = d.pop("preferences", UNSET)
        preferences: PreferencesPage | Unset
        if isinstance(_preferences,  Unset):
            preferences = UNSET
        else:
            preferences = PreferencesPage.from_dict(_preferences)




        preview_email = d.pop("previewEmail", UNSET)

        _subscribe = d.pop("subscribe", UNSET)
        subscribe: SubscribePage | Unset
        if isinstance(_subscribe,  Unset):
            subscribe = UNSET
        else:
            subscribe = SubscribePage.from_dict(_subscribe)




        _unsubscribe = d.pop("unsubscribe", UNSET)
        unsubscribe: UnsubscribePage | Unset
        if isinstance(_unsubscribe,  Unset):
            unsubscribe = UNSET
        else:
            unsubscribe = UnsubscribePage.from_dict(_unsubscribe)




        public_page_input = cls(
            archive=archive,
            confirm=confirm,
            hosted_form_id=hosted_form_id,
            preferences=preferences,
            preview_email=preview_email,
            subscribe=subscribe,
            unsubscribe=unsubscribe,
        )

        return public_page_input

