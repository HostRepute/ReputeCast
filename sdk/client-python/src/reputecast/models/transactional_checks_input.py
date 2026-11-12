from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.transactional_checks_input_editor_mode import TransactionalChecksInputEditorMode
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.transactional_test_data import TransactionalTestData
  from ..models.transactional_variable import TransactionalVariable





T = TypeVar("T", bound="TransactionalChecksInput")



@_attrs_define
class TransactionalChecksInput:
    """ 
        Attributes:
            editor_mode (TransactionalChecksInputEditorMode | Unset):
            html (str | Unset):
            mjml (str | Unset):
            plain_text (str | Unset):
            preheader (str | Unset):
            subject (str | Unset):
            test_data (list[TransactionalTestData] | Unset):
            track_clicks (bool | Unset):
            track_opens (bool | Unset):
            variables (list[TransactionalVariable] | Unset):
     """

    editor_mode: TransactionalChecksInputEditorMode | Unset = UNSET
    html: str | Unset = UNSET
    mjml: str | Unset = UNSET
    plain_text: str | Unset = UNSET
    preheader: str | Unset = UNSET
    subject: str | Unset = UNSET
    test_data: list[TransactionalTestData] | Unset = UNSET
    track_clicks: bool | Unset = UNSET
    track_opens: bool | Unset = UNSET
    variables: list[TransactionalVariable] | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.transactional_test_data import TransactionalTestData # noqa: PLC0415
        from ..models.transactional_variable import TransactionalVariable # noqa: PLC0415
        editor_mode: str | Unset = UNSET
        if not isinstance(self.editor_mode, Unset):
            editor_mode = self.editor_mode.value


        html = self.html

        mjml = self.mjml

        plain_text = self.plain_text

        preheader = self.preheader

        subject = self.subject

        test_data: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.test_data, Unset):
            test_data = []
            for test_data_item_data in self.test_data:
                test_data_item = test_data_item_data.to_dict()
                test_data.append(test_data_item)



        track_clicks = self.track_clicks

        track_opens = self.track_opens

        variables: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.variables, Unset):
            variables = []
            for variables_item_data in self.variables:
                variables_item = variables_item_data.to_dict()
                variables.append(variables_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if editor_mode is not UNSET:
            field_dict["editorMode"] = editor_mode
        if html is not UNSET:
            field_dict["html"] = html
        if mjml is not UNSET:
            field_dict["mjml"] = mjml
        if plain_text is not UNSET:
            field_dict["plainText"] = plain_text
        if preheader is not UNSET:
            field_dict["preheader"] = preheader
        if subject is not UNSET:
            field_dict["subject"] = subject
        if test_data is not UNSET:
            field_dict["testData"] = test_data
        if track_clicks is not UNSET:
            field_dict["trackClicks"] = track_clicks
        if track_opens is not UNSET:
            field_dict["trackOpens"] = track_opens
        if variables is not UNSET:
            field_dict["variables"] = variables

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.transactional_test_data import TransactionalTestData # noqa: PLC0415
        from ..models.transactional_variable import TransactionalVariable # noqa: PLC0415
        d = dict(src_dict)
        _editor_mode = d.pop("editorMode", UNSET)
        editor_mode: TransactionalChecksInputEditorMode | Unset
        if isinstance(_editor_mode,  Unset):
            editor_mode = UNSET
        else:
            editor_mode = TransactionalChecksInputEditorMode(_editor_mode)




        html = d.pop("html", UNSET)

        mjml = d.pop("mjml", UNSET)

        plain_text = d.pop("plainText", UNSET)

        preheader = d.pop("preheader", UNSET)

        subject = d.pop("subject", UNSET)

        _test_data = d.pop("testData", UNSET)
        test_data: list[TransactionalTestData] | Unset = UNSET
        if _test_data is not UNSET:
            test_data = []
            for test_data_item_data in _test_data:
                test_data_item = TransactionalTestData.from_dict(test_data_item_data)



                test_data.append(test_data_item)


        track_clicks = d.pop("trackClicks", UNSET)

        track_opens = d.pop("trackOpens", UNSET)

        _variables = d.pop("variables", UNSET)
        variables: list[TransactionalVariable] | Unset = UNSET
        if _variables is not UNSET:
            variables = []
            for variables_item_data in _variables:
                variables_item = TransactionalVariable.from_dict(variables_item_data)



                variables.append(variables_item)


        transactional_checks_input = cls(
            editor_mode=editor_mode,
            html=html,
            mjml=mjml,
            plain_text=plain_text,
            preheader=preheader,
            subject=subject,
            test_data=test_data,
            track_clicks=track_clicks,
            track_opens=track_opens,
            variables=variables,
        )

        return transactional_checks_input

